/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package new

import (
	"github.com/spf13/cobra"
)

type NewServiceInterface interface {
	CreatePJ(presetName string) error
}

func NewCmd(s NewServiceInterface) *cobra.Command {
	return &cobra.Command{
		Use:   "new [preset name]",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: プリセット名として有効な値の検証。

			presetName := args[0]

			return s.CreatePJ(presetName)
		},
	}
}
