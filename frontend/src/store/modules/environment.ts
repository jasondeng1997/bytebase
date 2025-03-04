import { defineStore } from "pinia";
import axios from "axios";
import { computed } from "vue";
import {
  empty,
  EMPTY_ID,
  Environment,
  EnvironmentCreate,
  EnvironmentId,
  EnvironmentPatch,
  EnvironmentState,
  ResourceObject,
  RowStatus,
  unknown,
} from "@/types";
import { usePolicyStore } from "./policy";
import { environmentName } from "../../utils";

function convert(
  environment: ResourceObject,
  includedList: ResourceObject[]
): Environment {
  return {
    ...(environment.attributes as Omit<Environment, "id">),
    id: parseInt(environment.id),
  };
}

export const useEnvironmentStore = defineStore("environment", {
  state: (): EnvironmentState => ({
    environmentList: [],
  }),
  actions: {
    convert(
      environment: ResourceObject,
      includedList: ResourceObject[]
    ): Environment {
      return convert(environment, includedList);
    },
    getEnvironmentList(rowStatusList?: RowStatus[]): Environment[] {
      return this.environmentList.filter((environment: Environment) => {
        return (
          (!rowStatusList && environment.rowStatus == "NORMAL") ||
          (rowStatusList && rowStatusList.includes(environment.rowStatus))
        );
      });
    },
    getEnvironmentById(environmentId: EnvironmentId): Environment {
      if (environmentId == EMPTY_ID) {
        return empty("ENVIRONMENT") as Environment;
      }

      for (const environment of this.environmentList) {
        if (environment.id == environmentId) {
          return environment;
        }
      }
      return unknown("ENVIRONMENT") as Environment;
    },
    getEnvironmentNameById(environmentId: EnvironmentId): string {
      const env = this.getEnvironmentById(environmentId);
      return environmentName(env);
    },
    upsertEnvironmentList(environmentList: Environment[]) {
      for (const environment of environmentList) {
        const i = this.environmentList.findIndex(
          (item: Environment) => item.id == environment.id
        );
        if (i != -1) {
          this.environmentList[i] = environment;
        } else {
          this.environmentList.push(environment);
        }

        this.environmentList.sort((a, b) => a.order - b.order);
      }
    },
    async fetchEnvironmentList(rowStatusList?: RowStatus[]) {
      const path =
        "/api/environment" +
        (rowStatusList ? "?rowstatus=" + rowStatusList.join(",") : "");
      const data = (await axios.get(path)).data;
      const environmentList: Environment[] = data.data.map(
        (env: ResourceObject) => {
          return convert(env, data.included);
        }
      );
      this.upsertEnvironmentList(environmentList);

      const policyStore = usePolicyStore();

      await Promise.all(
        environmentList.map((environment) => {
          return policyStore.fetchPolicyByEnvironmentAndType({
            environmentId: environment.id,
            type: "bb.policy.pipeline-approval",
          });
        })
      );

      return environmentList;
    },
    async createEnvironment(newEnvironment: EnvironmentCreate) {
      const data = (
        await axios.post(`/api/environment`, {
          data: {
            type: "environment",
            attributes: newEnvironment,
          },
        })
      ).data;
      const createdEnvironment = convert(data.data, data.included);

      this.upsertEnvironmentList([createdEnvironment]);

      await usePolicyStore().fetchPolicyByEnvironmentAndType({
        environmentId: createdEnvironment.id,
        type: "bb.policy.pipeline-approval",
      });

      return createdEnvironment;
    },
    async reorderEnvironmentList(orderedEnvironmentList: Environment[]) {
      const list: any[] = [];
      orderedEnvironmentList.forEach((item, index) => {
        list.push({
          // Server uses google/jsonapi which expects a string type for the special id field.
          // Afterwards, server will automatically serialize into int as declared by the EnvironmentPatch interface.
          id: item.id.toString(),
          type: "environmentPatch",
          attributes: {
            order: index,
          },
        });
      });
      const data = (
        await axios.patch(`/api/environment/reorder`, {
          data: list,
        })
      ).data;
      const environmentList: Environment[] = data.data.map(
        (env: ResourceObject) => {
          return convert(env, data.included);
        }
      );
      this.upsertEnvironmentList(environmentList);

      return environmentList;
    },
    async patchEnvironment({
      environmentId,
      environmentPatch,
    }: {
      environmentId: EnvironmentId;
      environmentPatch: EnvironmentPatch;
    }) {
      const data = (
        await axios.patch(`/api/environment/${environmentId}`, {
          data: {
            type: "environmentPatch",
            attributes: environmentPatch,
          },
        })
      ).data;
      const updatedEnvironment = convert(data.data, data.included);

      this.upsertEnvironmentList([updatedEnvironment]);

      return updatedEnvironment;
    },
  },
});

export const useEnvironmentList = (rowStatusList?: RowStatus[]) => {
  const store = useEnvironmentStore();
  return computed(() => store.getEnvironmentList(rowStatusList));
};
