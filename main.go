package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "cobra-cli",
		Short: "A CLI application with dynamic configuration reloading",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Server Port: %s\n", viper.GetString("server.port"))
			fmt.Printf("Server Debug: %v\n", viper.GetBool("server.debug"))
		},
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("COBRA")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	bindEnvVars()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: config file not loaded: %v", err)
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Config file changed: %s", e.Name)
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Error reloading config: %s", err)
			return
		}
		viper.AutomaticEnv()
		bindEnvVars()
	})
	viper.WatchConfig()
}

func bindEnvVars() {
	_ = viper.BindEnv("server.port", "COBRA_SERVER_PORT")
	_ = viper.BindEnv("server.debug", "COBRA_SERVER_DEBUG")
}
