/*
Copyright © 2021 StarForger <sparkforger@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"github.com/StarForger/neb-mc-rcon/cli"
	"github.com/spf13/cobra"	
	"github.com/spf13/viper"
	"net"
	homedir "github.com/mitchellh/go-homedir"
	"log"
)

var ( 
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "neb-mc-rcon [flags] [command ...]",
	Short: "CLI for RCON server interaction",
	Long: `CLI for interacting with RCON game servers.
	With no arguments, the CLI will run an interactive session 
	If arguments are included, they are sent as commands to the server.
	For example:

	rcon -H example.com 
	rcon -H minecraft.com stop
	RCON_PORT=25575 rcon list

`,
	
	Run: func(cmd *cobra.Command, args []string) { 
		ver := viper.GetBool("version")

		if ver {
			fmt.Fprintln(os.Stdout, "Version " + BuildVersion)
			return
		}

		host := viper.GetString("host")
		port := viper.GetString("port")
		pwd := viper.GetString("password")

		uri := net.JoinHostPort(host, port)

		if len(args) == 0 {
			cli.Run(uri, pwd, os.Stdin, os.Stdout)
		} else {
			cli.Execute(uri, pwd, os.Stdout, args...)
		}
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rcon.yml)")
	rootCmd.PersistentFlags().StringP("host", "H", "localhost", "RCON server's hostname")
	rootCmd.PersistentFlags().String("password", "", "RCON server's password")
	rootCmd.PersistentFlags().Int("port", 25575, "RCON port")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "version number")
	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		log.Fatal(err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".rcon" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".rcon")
		viper.SetConfigType("yml")
	}
	viper.SetEnvPrefix("rcon")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
