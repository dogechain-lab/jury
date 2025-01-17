package backup

import (
	"github.com/dogechain-lab/dogechain/command"
	"github.com/spf13/cobra"

	"github.com/dogechain-lab/dogechain/command/helper"
)

func GetCommand() *cobra.Command {
	backupCmd := &cobra.Command{
		Use:     "backup",
		Short:   "Create blockchain backup file by fetching blockchain data from the running node",
		PreRunE: runPreRun,
		Run:     runCommand,
	}

	helper.RegisterGRPCAddressFlag(backupCmd)

	setFlags(backupCmd)
	helper.SetRequiredFlags(backupCmd, params.getRequiredFlags())

	return backupCmd
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&params.out,
		outFlag,
		"",
		"the export path for the backup",
	)

	cmd.Flags().StringVar(
		&params.fromRaw,
		fromFlag,
		"0",
		"the beginning height of the chain in backup",
	)

	cmd.Flags().StringVar(
		&params.toRaw,
		toFlag,
		"",
		"the end height of the chain in backup",
	)

	cmd.Flags().BoolVar(
		&params.overwriteFile,
		overwriteFileFlag,
		false,
		"force overwrite the backup file if it already exists",
	)

	cmd.Flags().BoolVar(
		&params.enableZstdCompression,
		zstdFlag,
		false,
		"enable zstd compression",
	)

	cmd.Flags().IntVar(
		&params.zstdLevel,
		zstdLevelFlag,
		3,
		"zstd compression level, range 1-10",
	)
}

func runPreRun(_ *cobra.Command, _ []string) error {
	return params.validateFlags()
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	if err := params.createBackup(helper.GetGRPCAddress(cmd)); err != nil {
		outputter.SetError(err)

		return
	}

	outputter.SetCommandResult(params.getResult())
}
