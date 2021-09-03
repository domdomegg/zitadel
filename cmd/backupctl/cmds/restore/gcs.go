package restore

import (
	"context"
	"github.com/caos/orbos/mntr"
	"github.com/caos/zitadel/cmd/backupctl/cmds/helpers"
	"github.com/caos/zitadel/pkg/backup"
	"github.com/spf13/cobra"
)

func GCSCommand(ctx context.Context, monitor mntr.Monitor) *cobra.Command {
	var (
		backupName       string
		backupNameEnv    string
		assetEndpoint    string
		assetAKID        string
		assetSAK         string
		sourceBucket     string
		sourceSAJSONPath string
		configPath       string
		certsDir         string
		host             string
		port             string
		cmd              = &cobra.Command{
			Use:   "gcs",
			Short: "Restore from GCS Bucket",
			Long:  "Restore from GCS Bucket",
		}
	)

	flags := cmd.Flags()
	flags.StringVar(&backupName, "backupname", "", "Backupname used in destination file path")
	flags.StringVar(&backupNameEnv, "backupnameenv", "", "Backupnameenv used in destination file path")
	flags.StringVar(&assetEndpoint, "asset-endpoint", "", "Endpoint for the asset S3 storage")
	flags.StringVar(&assetAKID, "asset-akid", "", "AccessKeyID for the asset S3 storage")
	flags.StringVar(&assetSAK, "asset-sak", "", "SecretAccessKey for the asset S3 storage")
	flags.StringVar(&sourceSAJSONPath, "source-sajsonpath", "~/sa.json", "Path to where ServiceAccount-json will be written for the source GCS")
	flags.StringVar(&sourceBucket, "source-bucket", "", "Bucketname in the source GCS")
	flags.StringVar(&configPath, "configpath", "~/rsync.conf", "Path used to save rsync configuration")
	flags.StringVar(&certsDir, "certs-dir", "", "Folder with certificates used to connect to cockroachdb")
	flags.StringVar(&host, "host", "", "Host used to connect to cockroachdb")
	flags.StringVar(&port, "port", "", "Port used to connect to cockroachdb")

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {

		if err := helpers.ValidateBackupFlags(
			backupName,
			backupNameEnv,
		); err != nil {
			return err
		}

		if err := helpers.ValidateGCSFlags(
			sourceSAJSONPath,
			sourceBucket,
		); err != nil {
			return err
		}

		if err := helpers.ValidateSourceS3Flags(
			assetEndpoint,
			assetAKID,
			assetSAK,
			"notempty",
		); err != nil {
			return err
		}

		if err := helpers.ValidateCockroachFlags(
			certsDir,
			host,
			port,
		); err != nil {
			return err
		}

		if err := backup.RsyncRestoreGCSToS3(
			ctx,
			backupName,
			backupNameEnv,
			"destination",
			assetEndpoint,
			assetAKID,
			assetSAK,
			"source",
			sourceSAJSONPath,
			sourceBucket,
			configPath,
		); err != nil {
			return err
		}

		if err := backup.CockroachRestoreFromGCS(
			ctx,
			certsDir,
			sourceBucket,
			backupName,
			backupNameEnv,
			host,
			port,
			sourceSAJSONPath,
		); err != nil {
			return err
		}

		return nil
	}
	return cmd
}
