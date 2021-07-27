package cmd

import (
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
				if err := fetch(url); err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func fetch(s string) error {
	log.Debug().Msgf("url: %s", s)
	res, err := http.Get(s)
	cobra.CheckErr(err)
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	cobra.CheckErr(err)

	u, err := url.Parse(s)
	cobra.CheckErr(err)

	return ioutil.WriteFile(fmt.Sprintf("%s.html", u.Hostname()), body, 0644)
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
	u, err := url.Parse(s)
	cobra.CheckErr(err)
	return u.Hostname(), nil
}

func lastFetch(s string) (time.Time, error) {
	site, err := site(s)
	cobra.CheckErr(err)
	f, err := os.Open(fmt.Sprintf("%s.html", site))
	cobra.CheckErr(err)
	fi, err := f.Stat()
	cobra.CheckErr(err)

	return fi.ModTime(), nil
}

func numLinks(s string) (int, error) {
	var out int
	site, err := site(s)
	cobra.CheckErr(err)
	f, err := os.Open(fmt.Sprintf("%s.html", site))
	cobra.CheckErr(err)
	doc, err := html.Parse(f)
	cobra.CheckErr(err)
	var bfs func(*html.Node)
	bfs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					log.Debug().Msgf("a: %v", a.Val)
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

func images(s string) (int, error) {
	var out int
	site, err := site(s)
	cobra.CheckErr(err)
	f, err := os.Open(fmt.Sprintf("%s.html", site))
	cobra.CheckErr(err)
	doc, err := html.Parse(f)
	cobra.CheckErr(err)
	var bfs func(*html.Node)
	bfs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, a := range n.Attr {
				if a.Key == "src" {
					log.Debug().Msgf("img: %v", a.Val)
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
