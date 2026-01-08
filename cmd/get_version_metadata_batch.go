package cmd

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/majd/ipatool/v2/pkg/appstore"
	"github.com/spf13/cobra"
)

type ExternalVersionIdList struct {
	AppVerIDs []string `json:"appVerIDs"`
}

// nolint:wrapcheck
func getVersionMetadataBatchCmd() *cobra.Command {
	var (
		appID              int64
		bundleID           string
		externalVersionIDs string
	)

	cmd := &cobra.Command{
		Use:   "get-version-metadata-batch",
		Short: "Retrieves the metadata for a specific version of an app",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appID == 0 && bundleID == "" {
				return errors.New("either the app ID or the bundle identifier must be specified")
			}

			var lastErr error
			var acc appstore.Account

			return retry.Do(func() error {
				infoResult, err := dependencies.AppStore.AccountInfo()
				if err != nil {
					return err
				}

				acc = infoResult.Account

				if errors.Is(lastErr, appstore.ErrPasswordTokenExpired) {
					loginResult, err := dependencies.AppStore.Login(appstore.LoginInput{Email: acc.Email, Password: acc.Password})
					if err != nil {
						return err
					}

					acc = loginResult.Account
				}

				app := appstore.App{ID: appID}
				if bundleID != "" {
					lookupResult, err := dependencies.AppStore.Lookup(appstore.LookupInput{Account: acc, BundleID: bundleID})
					if err != nil {
						return err
					}

					app = lookupResult.App
				}

				dependencies.Logger.Log().
					Str("externalVersionIDs", externalVersionIDs)

				var externalVersionIdList ExternalVersionIdList
				err = json.Unmarshal([]byte(externalVersionIDs), &externalVersionIdList)
				if err != nil {
					return err
				}

				sem := make(chan struct{}, 8) // 最多同时 8 个
				var wg sync.WaitGroup
				for _, id := range externalVersionIdList.AppVerIDs {
					id := id
					wg.Add(1)
					go func() {
						defer wg.Done()
						sem <- struct{}{}        // 获取名额
						defer func() { <-sem }() // 归还名额

						out, err := dependencies.AppStore.GetVersionMetadata(appstore.GetVersionMetadataInput{
							Account:   acc,
							App:       app,
							VersionID: id,
						})
						if err != nil {
							return
						}

						dependencies.Logger.Log().
							Str("externalVersionID", id).
							Str("displayVersion", out.DisplayVersion).
							Time("releaseDate", out.ReleaseDate).
							Bool("success", true).
							Send()
					}()
				}
				wg.Wait()
				return nil
			},
				retry.LastErrorOnly(true),
				retry.DelayType(retry.FixedDelay),
				retry.Delay(time.Millisecond),
				retry.Attempts(2),
				retry.RetryIf(func(err error) bool {
					lastErr = err

					return errors.Is(err, appstore.ErrPasswordTokenExpired)
				}),
			)
		},
	}

	cmd.Flags().Int64VarP(&appID, "app-id", "i", 0, "ID of the target iOS app (required)")
	cmd.Flags().StringVarP(&bundleID, "bundle-identifier", "b", "", "The bundle identifier of the target iOS app (overrides the app ID)")
	cmd.Flags().StringVar(&externalVersionIDs, "external-version-id-list", "", "External version identifier of the target iOS app (required)")

	_ = cmd.MarkFlagRequired("external-version-id-list")

	return cmd
}
