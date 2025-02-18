/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/brutella/dnssd"
	"github.com/lab42/kdns/handler"
	"github.com/lab42/kdns/mdns"
	"github.com/lab42/kdns/watcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kdns",
	Short: "A brief description of your application",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// Set up a channel to listen for termination signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		mDNSManager, err := mdns.NewManager()
		if err != nil {
			return err
		}

		mDNSManager.Upsert(dnssd.Config{
			Name:   "k3s", // This will be the hostname
			Host:   "k3s",
			Type:   "_ssh._tcp",
			Domain: "local",
			Port:   22,
		})

		ingressHandler := handler.NewIngressHandler(&mDNSManager)
		serviceHandler := handler.NewServiceHandler(&mDNSManager)

		// Create the Kubernetes watcher
		watcher, err := watcher.NewK8sWatcher(ingressHandler, serviceHandler)
		if err != nil {
			return err
		}

		// Start watching for events
		go watcher.Run()
		go mDNSManager.Respond(context.Background())

		// Wait for a termination signal
		<-sigChan

		// Gracefully shutdown all services
		log.Println("shutting down ingress watcher...")
		watcher.Stop()
		log.Println("ingress watcher stopped.")

		return err
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kdns.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".kdns" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kdns")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
