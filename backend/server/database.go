package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser"
	"github.com/bytebase/bytebase/backend/plugin/parser/edit"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerDatabaseRoutes(g *echo.Group) {
	g.GET("/database", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseFind := new(api.DatabaseFind)
		if instanceIDStr := c.QueryParam("instance"); instanceIDStr != "" {
			instanceID, err := strconv.Atoi(instanceIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter instance is not a number: %s", instanceIDStr)).SetInternal(err)
			}
			databaseFind.InstanceID = &instanceID
		}
		if name := c.QueryParam("name"); name != "" {
			databaseFind.Name = &name
		}
		if syncStatusStr := c.QueryParam("syncStatus"); syncStatusStr != "" {
			syncStatus := api.SyncStatus(syncStatusStr)
			databaseFind.SyncStatus = &syncStatus
		}
		projectIDStr := c.QueryParams().Get("project")
		if projectIDStr != "" {
			projectID, err := strconv.Atoi(projectIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project query parameter is not a number: %s", projectIDStr)).SetInternal(err)
			}
			databaseFind.ProjectID = &projectID
		}
		dbList, err := s.store.FindDatabase(ctx, databaseFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch database list").SetInternal(err)
		}

		var filteredList []*api.Database
		role := c.Get(getRoleContextKey()).(api.Role)
		// If the caller is a developer, we will only return databases belonging to the
		// project where the caller is a member of.
		if role == api.Developer {
			principalID := c.Get(getPrincipalIDContextKey()).(int)
			for _, database := range dbList {
				policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &database.ProjectID})
				if err != nil {
					return err
				}
				if hasActiveProjectMembership(principalID, policy) {
					filteredList = append(filteredList, database)
				}
			}
		} else {
			filteredList = dbList
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, filteredList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		composedDatabase, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database with ID %d", id)).SetInternal(err)
		}
		if composedDatabase == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}
		// Wildcard(*) database is used to connect all database at instance level.
		// Do not return it via `get database by id` API.
		if composedDatabase.Name == api.AllDatabaseName {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database with ID %d is a wildcard *", id))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedDatabase); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/database/:databaseID", func(c echo.Context) error {
		ctx := c.Request().Context()
		updaterID := c.Get(getPrincipalIDContextKey()).(int)
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get database with ID %d", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}
		oldProject, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
		if err != nil {
			return err
		}
		if oldProject == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("project%q not found", database.ProjectID))
		}

		dbPatch := &api.DatabasePatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, dbPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch database request").SetInternal(err)
		}

		var toProject *store.ProjectMessage
		if dbPatch.ProjectID != nil && *dbPatch.ProjectID != oldProject.UID {
			// Before updating database's projectID, we need to check if there are still bound sheets.
			sheetList, err := s.store.FindSheet(ctx, &api.SheetFind{DatabaseID: &database.UID}, updaterID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sheets by database %q", database.DatabaseName)).SetInternal(err)
			}
			if len(sheetList) > 0 {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("The transferring database has %d bound sheets, please go to SQL editor to unbind them first", len(sheetList)))
			}

			project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{UID: dbPatch.ProjectID})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find project with ID %d", *dbPatch.ProjectID)).SetInternal(err)
			}
			if project == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", *dbPatch.ProjectID))
			}
			if project.Deleted {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project %q is deleted", *dbPatch.ProjectID))
			}
			toProject = project
		}

		updateMessage := &store.UpdateDatabaseMessage{
			EnvironmentID: database.EnvironmentID,
			InstanceID:    database.InstanceID,
			DatabaseName:  database.DatabaseName,
		}
		if toProject != nil {
			updateMessage.ProjectID = &toProject.ResourceID
		}
		// Patch database labels
		// We will completely replace the old labels with the new ones, except bb.environment is immutable and
		// must match instance environment.
		if dbPatch.Labels != nil {
			labels := make(map[string]string)
			databaseLabels, err := convertDatabaseLabels(*dbPatch.Labels)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to set database labels").SetInternal(err)
			}
			for _, databaseLabel := range databaseLabels {
				labels[databaseLabel.Key] = databaseLabel.Value
			}
			updateMessage.Labels = &labels
		}
		updatedDatabase, err := s.store.UpdateDatabase(ctx, updateMessage, updaterID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
		}

		// If we are transferring the database to a different project, then we create a project activity in both
		// the old project and new project.
		if toProject != nil {
			if err := createTransferProjectActivity(ctx, s.store, updatedDatabase, oldProject, toProject, updaterID); err != nil {
				log.Error("failed to create project transfer activity", zap.Error(err))
			}
		}

		composedDatabase, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &database.UID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get database with ID %q", id)).SetInternal(err)
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedDatabase); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal database ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	// When query metadata is present, we will return the schema metadata. Otherwise, we will return the raw dump.
	g.GET("/database/:databaseID/schema", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database %q", database.DatabaseName)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		dbSchema, err := s.store.GetDBSchema(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get dbSchema for database %q", database.DatabaseName)).SetInternal(err)
		}
		if dbSchema == nil {
			// TODO(d): make SyncDatabaseSchema return the updated database schema.
			if err := s.SchemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database schema for database %q", database.DatabaseName)).SetInternal(err)
			}
			newDBSchema, err := s.store.GetDBSchema(ctx, id)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get dbSchema for database %q", database.DatabaseName)).SetInternal(err)
			}
			if newDBSchema == nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("New dbSchema not found for database %q", database.DatabaseName)).SetInternal(err)
			}
			dbSchema = newDBSchema
		}

		isQuerySchema := c.QueryParam("metadata") == ""
		if isQuerySchema {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)
			if _, err := c.Response().Write([]byte(dbSchema.Schema)); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to write schema response for database %q", database.DatabaseName)).SetInternal(err)
			}
		} else {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
			metadataBytes, err := protojson.Marshal(dbSchema.Metadata)
			if err != nil {
				return err
			}
			if _, err := c.Response().Write(metadataBytes); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to write schema response for database %q", database.DatabaseName)).SetInternal(err)
			}
		}
		return nil
	})

	g.POST("/database/:databaseID/backup", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		backupCreate := &api.BackupCreate{
			CreatorID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, backupCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create backup request").SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database %q", database.DatabaseName)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
		if err != nil {
			return err
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("instance %q not found", database.InstanceID))
		}
		if instance.Deleted {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("instance %q deleted", database.InstanceID))
		}
		if instance.Engine == db.MongoDB {
			return echo.NewHTTPError(http.StatusBadRequest, "Backup is not supported for MongoDB")
		}

		storeBackupList, err := s.store.FindBackup(ctx, &api.BackupFind{
			DatabaseID: &id,
			Name:       &backupCreate.Name,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch backup with name %q", backupCreate.Name)).SetInternal(err)
		}
		if len(storeBackupList) > 0 {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Backup %q already exists", backupCreate.Name))
		}

		backup, err := s.BackupRunner.ScheduleBackupTask(ctx, database, backupCreate.Name, backupCreate.Type, c.Get(getPrincipalIDContextKey()).(int))
		if err != nil {
			if common.ErrorCode(err) == common.DbConnectionFailure {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to connect to instance %q", database.InstanceID)).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to schedule task for backup %q", backupCreate.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backup); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create backup response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/backup", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("database %d not found", id)).SetInternal(err)
		}

		backupFind := &api.BackupFind{
			DatabaseID: &id,
		}
		backupList, err := s.store.FindBackup(ctx, backupFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to backup list for database id: %d", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch backup list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/database/:databaseID/backup-setting", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		backupSettingUpsert := &api.BackupSettingUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, backupSettingUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed set backup setting request").SetInternal(err)
		}
		backupSettingUpsert.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("database %d not found", id)).SetInternal(err)
		}
		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EnvironmentID})
		if err != nil {
			return err
		}
		backupSettingUpsert.EnvironmentID = environment.UID

		backupSetting, err := s.store.UpsertBackupSetting(ctx, backupSettingUpsert)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid backup setting").SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to set backup setting").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupSetting); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create set backup setting response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/backup-setting", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("database %d not found", id)).SetInternal(err)
		}

		backupSetting, err := s.store.GetBackupSettingByDatabaseID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get backup setting for database id: %d", id)).SetInternal(err)
		}
		if backupSetting == nil {
			// Returns the backup setting with UNKNOWN_ID to indicate the database has no backup
			backupSetting = &api.BackupSetting{
				ID: api.UnknownID,
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupSetting); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get backup setting response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/data-source/:dataSourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}
		dataSourceID, err := strconv.Atoi(c.Param("dataSourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Data source ID is not a number: %s", c.Param("dataSourceID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID, IncludeAllDatabase: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", databaseID)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get instance").SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance not found for database ID %d", databaseID))
		}

		composedInstance, err := s.store.GetInstanceByID(ctx, instance.UID)
		if err != nil {
			return err
		}
		var composedDataSource *api.DataSource
		for _, ds := range composedInstance.DataSourceList {
			if ds.ID == dataSourceID {
				composedDataSource = ds
				break
			}
		}
		if composedDataSource == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "data source not found")
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedDataSource); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal find data source response").SetInternal(err)
		}
		return nil
	})

	g.POST("/database/:databaseID/data-source", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID, IncludeAllDatabase: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", databaseID)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get instance").SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance not found for database ID %d", databaseID))
		}

		dataSourceCreate := &api.DataSourceCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, dataSourceCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create data source request").SetInternal(err)
		}
		if !s.licenseService.IsFeatureEnabled(api.FeatureReadReplicaConnection) && dataSourceCreate.Type == api.RO {
			if dataSourceCreate.Host != "" || dataSourceCreate.Port != "" {
				return echo.NewHTTPError(http.StatusForbidden, api.FeatureReadReplicaConnection.AccessErrorMessage())
			}
		}

		creatorID := c.Get(getPrincipalIDContextKey()).(int)
		title := api.AdminDataSourceName
		if dataSourceCreate.Type == api.RO {
			title = api.ReadOnlyDataSourceName
		}
		dataSourceMessage := &store.DataSourceMessage{
			Title:                  title,
			Type:                   dataSourceCreate.Type,
			Username:               dataSourceCreate.Username,
			ObfuscatedPassword:     common.Obfuscate(dataSourceCreate.Password, s.secret),
			ObfuscatedSslCa:        common.Obfuscate(dataSourceCreate.SslCa, s.secret),
			ObfuscatedSslCert:      common.Obfuscate(dataSourceCreate.SslCert, s.secret),
			ObfuscatedSslKey:       common.Obfuscate(dataSourceCreate.SslKey, s.secret),
			Host:                   dataSourceCreate.Host,
			Port:                   dataSourceCreate.Port,
			Database:               dataSourceCreate.Database,
			SRV:                    dataSourceCreate.Options.SRV,
			AuthenticationDatabase: dataSourceCreate.Options.AuthenticationDatabase,
		}
		if err := s.store.AddDataSourceToInstanceV2(ctx, instance.UID, creatorID, instance.EnvironmentID, instance.ResourceID, dataSourceMessage); err != nil {
			return err
		}

		composedInstance, err := s.store.GetInstanceByID(ctx, instance.UID)
		if err != nil {
			return err
		}
		var composedDataSource *api.DataSource
		for _, ds := range composedInstance.DataSourceList {
			if ds.Type == dataSourceCreate.Type {
				composedDataSource = ds
				break
			}
		}
		if composedDataSource == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "data source not found")
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedDataSource); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create data source response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/database/:databaseID/data-source/:dataSourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		dataSourceID, err := strconv.Atoi(c.Param("dataSourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Data source ID is not a number: %s", c.Param("dataSourceID"))).SetInternal(err)
		}

		// Because data source could use a wildcard database "*" as its database,
		// so we need to include wildcard databases when check if relevant database exists.
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID, IncludeAllDatabase: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", databaseID)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get instance").SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance not found for database ID %d", databaseID))
		}
		var dataSource *store.DataSourceMessage
		for _, ds := range instance.DataSources {
			if ds.UID == dataSourceID {
				dataSource = ds
				break
			}
		}
		if dataSource == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("data source not found by ID %d and database ID %d", dataSourceID, databaseID))
		}

		dataSourcePatch := &api.DataSourcePatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, dataSourcePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch data source request").SetInternal(err)
		}
		if dataSource.Type == api.RO && !s.licenseService.IsFeatureEnabled(api.FeatureReadReplicaConnection) {
			if (dataSourcePatch.Host != nil && *dataSourcePatch.Host != "") || (dataSourcePatch.Port != nil && *dataSourcePatch.Port != "") {
				return echo.NewHTTPError(http.StatusForbidden, api.FeatureReadReplicaConnection.AccessErrorMessage())
			}
		}

		updateMessage := &store.UpdateDataSourceMessage{
			UpdaterID:     c.Get(getPrincipalIDContextKey()).(int),
			InstanceUID:   instance.UID,
			EnvironmentID: instance.EnvironmentID,
			InstanceID:    instance.ResourceID,
			Type:          dataSource.Type,
			Username:      dataSourcePatch.Username,
			Host:          dataSourcePatch.Host,
			Port:          dataSourcePatch.Port,
		}
		if dataSourcePatch.Password != nil {
			obfuscated := common.Obfuscate(*dataSourcePatch.Password, s.secret)
			updateMessage.ObfuscatedPassword = &obfuscated
		}
		if dataSourcePatch.SslCa != nil {
			obfuscated := common.Obfuscate(*dataSourcePatch.SslCa, s.secret)
			updateMessage.ObfuscatedSslCa = &obfuscated
		}
		if dataSourcePatch.SslCert != nil {
			obfuscated := common.Obfuscate(*dataSourcePatch.SslCert, s.secret)
			updateMessage.ObfuscatedSslCert = &obfuscated
		}
		if dataSourcePatch.SslKey != nil {
			obfuscated := common.Obfuscate(*dataSourcePatch.SslKey, s.secret)
			updateMessage.ObfuscatedSslKey = &obfuscated
		}
		if dataSourcePatch.UseEmptyPassword != nil && *dataSourcePatch.UseEmptyPassword {
			obfuscated := common.Obfuscate("", s.secret)
			updateMessage.ObfuscatedPassword = &obfuscated
		}

		if dataSourcePatch.Options != nil {
			updateMessage.SRV = &dataSourcePatch.Options.SRV
			updateMessage.AuthenticationDatabase = &dataSourcePatch.Options.AuthenticationDatabase
		}
		if err := s.store.UpdateDataSourceV2(ctx, updateMessage); err != nil {
			return err
		}

		if dataSource.Type == api.Admin {
			if _, err := s.SchemaSyncer.SyncInstance(ctx, instance); err != nil {
				log.Warn("Failed to sync instance",
					zap.String("instance", instance.ResourceID),
					zap.Error(err))
			}
			// Sync all databases in the instance asynchronously.
			updatedInstance, err := s.store.GetInstanceByID(ctx, instance.UID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated instance %q", instance.Title)).SetInternal(err)
			}
			s.stateCfg.InstanceDatabaseSyncChan <- updatedInstance
		}

		composedInstance, err := s.store.GetInstanceByID(ctx, instance.UID)
		if err != nil {
			return err
		}
		var composedDataSource *api.DataSource
		for _, ds := range composedInstance.DataSourceList {
			if ds.Type == dataSource.Type {
				composedDataSource = ds
				break
			}
		}
		if composedDataSource == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "data source not found")
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedDataSource); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal patch data source response").SetInternal(err)
		}
		return nil
	})

	g.DELETE("/database/:databaseID/data-source/:dataSourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		dataSourceID, err := strconv.Atoi(c.Param("dataSourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Data source ID is not a number: %s", c.Param("dataSourceID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID, IncludeAllDatabase: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", databaseID)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get instance").SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance not found for database ID %d", databaseID))
		}

		var dataSource *store.DataSourceMessage
		for _, ds := range instance.DataSources {
			if ds.UID == dataSourceID {
				dataSource = ds
				break
			}
		}
		if dataSource == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("data source not found by ID %d and database ID %d", dataSourceID, databaseID))
		}
		if dataSource.Type == api.Admin {
			return echo.NewHTTPError(http.StatusBadRequest, "admin data source cannot be deleted")
		}
		if err := s.store.RemoveDataSourceV2(ctx, instance.UID, instance.EnvironmentID, instance.ResourceID, dataSource.Type); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.POST("/database/:databaseID/edit", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get database").SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{EnvironmentID: &database.EnvironmentID, ResourceID: &database.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get instance").SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance not found for database ID %d", databaseID))
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
		}
		databaseEdit := &api.DatabaseEdit{}
		if err := json.Unmarshal(body, databaseEdit); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed post database edit request").SetInternal(err)
		}

		engineType := parser.EngineType(instance.Engine)
		validateResultList, err := edit.ValidateDatabaseEdit(engineType, databaseEdit)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate DatabaseEdit").SetInternal(err)
		}

		databaseEditResult := &api.DatabaseEditResult{
			Statement:          "",
			ValidateResultList: validateResultList,
		}
		if len(validateResultList) == 0 {
			statement, err := edit.DeparseDatabaseEdit(engineType, databaseEdit)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to deparse DatabaseEdit").SetInternal(err)
			}
			databaseEditResult.Statement = statement
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, databaseEditResult); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database edit result response").SetInternal(err)
		}
		return nil
	})
}

func createTransferProjectActivity(ctx context.Context, stores *store.Store, database *store.DatabaseMessage, oldProject, newProject *store.ProjectMessage, updaterID int) error {
	existingProject, err := stores.GetProjectV2(ctx, &store.FindProjectMessage{UID: &oldProject.UID})
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(api.ActivityProjectDatabaseTransferPayload{
		DatabaseID:   database.UID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return err
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   updaterID,
		ContainerID: oldProject.UID,
		Type:        api.ActivityProjectDatabaseTransfer,
		Level:       api.ActivityInfo,
		Comment:     fmt.Sprintf("Transferred out database %q to project %q.", database.DatabaseName, newProject.Title),
		Payload:     string(bytes),
	}
	if _, err := stores.CreateActivity(ctx, activityCreate); err != nil {
		log.Warn("Failed to create project activity after transferring database",
			zap.Int("database_id", database.UID),
			zap.String("database_name", database.DatabaseName),
			zap.Int("old_project_id", oldProject.UID),
			zap.Int("new_project_id", newProject.UID),
			zap.Error(err))
	}

	activityCreate = &api.ActivityCreate{
		CreatorID:   updaterID,
		ContainerID: newProject.UID,
		Type:        api.ActivityProjectDatabaseTransfer,
		Level:       api.ActivityInfo,
		Comment:     fmt.Sprintf("Transferred in database %q from project %q.", database.DatabaseName, existingProject.Title),
		Payload:     string(bytes),
	}
	if _, err := stores.CreateActivity(ctx, activityCreate); err != nil {
		log.Warn("Failed to create project activity after transferring database",
			zap.Int("database_id", database.UID),
			zap.String("database_name", database.DatabaseName),
			zap.Int("old_project_id", oldProject.UID),
			zap.Int("new_project_id", newProject.UID),
			zap.Error(err))
	}
	return nil
}
