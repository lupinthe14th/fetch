package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/html"
)

var (
	cfgFile  string
	urls     []string
	debug    bool
	metadata bool
)

type Metadata struct {
	Site      string    `json:"site"`
	NumLinks  int       `json:"num_links"`
	Images    int       `json:"images"`
	LastFetch time.Time `json:"last_fetch"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fetch",
	Short: "fetch is be able to web page and saves them to diks for later retrival and browsing for command line program.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires a urls arguments")
		}
		urls = args
		return nil
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug().Msgf("urls: %v", urls)
		switch {
		case metadata:
			for _, url := range urls {
				if err := fetchMetadata(url); err != nil {
					return err
				}
			}
		default:
			for _, url := range urls {
				if err := fetch(context.Background(), url); err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func fetch(ctx context.Context, s string) error {
	log.Debug().Msgf("url: %s", s)
	u, err := url.Parse(s)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fmt.Sprintf("%s.html", u.Hostname()), body, 0600)
}

func fetchMetadata(s string) error {
	var (
		metadata Metadata
		err      error
	)

	// site
	metadata.Site, err = site(s)
	cobra.CheckErr(err)
	// last fetch
	metadata.LastFetch, err = lastFetch(s)
	cobra.CheckErr(err)
	// num links
	metadata.NumLinks, err = numLinks(s)
	cobra.CheckErr(err)
	// images
	metadata.Images, err = images(s)
	cobra.CheckErr(err)

	b, err := json.Marshal(metadata)
	cobra.CheckErr(err)
	fmt.Println(string(b))
	return nil
}

func site(s string) (string, error) {
	var out string
	u, err := url.Parse(s)
	if err != nil {
		return out, err
	}
	out = u.Hostname()
	return out, nil
}

func lastFetch(s string) (time.Time, error) {
	var out time.Time
	site, err := site(s)
	if err != nil {
		return out, err
	}
	f, err := os.Open(fmt.Sprintf("%s.html", site))
	if err != nil {
		return out, err
	}
	fi, err := f.Stat()
	if err != nil {
		return out, err
	}
	out = fi.ModTime()
	return out, nil
}

func numLinks(s string) (int, error) {
	return bfs(s, "a", "href")
}

func images(s string) (int, error) {
	return bfs(s, "img", "src")
}

func bfs(s, data, key string) (int, error) {
	var out int
	site, err := site(s)
	if err != nil {
		return out, err
	}
	f, err := os.Open(fmt.Sprintf("%s.html", site))
	if err != nil {
		return out, err
	}
	doc, err := html.Parse(f)
	if err != nil {
		return out, nil
	}
	var bfs func(*html.Node)
	bfs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == data {
			for _, a := range n.Attr {
				if a.Key == key {
					log.Debug().Msgf("%s: %s", data, a.Val)
					out++
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			bfs(c)
		}
	}
	bfs(doc)
	return out, nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug mode")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fetch.yaml)")

	rootCmd.Flags().BoolVarP(&metadata, "metadata", "m", false, "Record metadata about what was fetched.")
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

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".fetch")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
