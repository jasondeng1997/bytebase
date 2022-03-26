package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

var (
	// Some settings contain secret info so we only return settings that are needed by the client.
	whitelistSettings = []api.SettingName{
		api.SettingBrandingLogo,
	}
)

func (s *Server) registerSettingRoutes(g *echo.Group) {
	g.GET("/setting", func(c echo.Context) error {
		ctx := context.Background()
		find := &api.SettingFind{}
		settingRawList, err := s.SettingService.FindSettingList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch setting list").SetInternal(err)
		}
		var settingList []*api.Setting
		for _, raw := range settingRawList {
			setting, err := s.composeSettingRelationship(ctx, raw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose setting relationship for ID %d", raw.ID)).SetInternal(err)
			}
			settingList = append(settingList, setting)
		}

		filteredList := []*api.Setting{}
		for _, setting := range settingList {
			for _, whitelist := range whitelistSettings {
				if setting.Name == whitelist {
					filteredList = append(filteredList, setting)
					break
				}
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, filteredList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal project list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/setting/:name", func(c echo.Context) error {
		ctx := context.Background()
		settingPatch := &api.SettingPatch{
			Name:      api.SettingName(c.Param("name")),
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}

		if settingPatch.Name == api.SettingBrandingLogo && !s.feature(api.FeatureBranding) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureBranding.AccessErrorMessage())
		}

		if err := jsonapi.UnmarshalPayload(c.Request().Body, settingPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update setting request").SetInternal(err)
		}

		setting, err := s.SettingService.PatchSetting(ctx, settingPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Setting name not found: %s", settingPatch.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update setting: %v", settingPatch.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, setting.ToSetting()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal setting response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeSettingRelationship(ctx context.Context, raw *api.SettingRaw) (*api.Setting, error) {
	setting := raw.ToSetting()

	creator, err := s.store.GetPrincipalByID(ctx, setting.CreatorID)
	if err != nil {
		return nil, err
	}
	setting.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, setting.UpdaterID)
	if err != nil {
		return nil, err
	}
	setting.Updater = updater

	return setting, nil
}
