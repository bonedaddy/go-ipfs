package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	iCore "github.com/ipfs/interface-go-ipfs-core"
	iCorePath "github.com/ipfs/interface-go-ipfs-core/path"
)

type cfgOpt func(*config.Config)

func setupPlugins(path string) error {
	// Load plugins. This will skip the repo if not available.
	plugins, err := loader.NewPluginLoader(filepath.Join(path, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func createTempRepo(ctx context.Context) (string, error) {
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Set default config with option for 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return "", err
	}

	// configure the temporary node
	// cfg.Routing.Type = "dhtclient"
	// cfg.Experimental.QUIC = true
	cfg.Datastore.Spec = map[string]interface{}{
		"type": "mem",
		"path": "blocks",
	}

	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

/// ------ Spawning the node
func createNode(ctx context.Context, repoPath string) (iCore.CoreAPI, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node
	node, err := core.NewNode(ctx, &core.BuildCfg{
		Online: true,
		// Routing: libp2p.DHTClientOption,
		Repo: repo,
	})
	if err != nil {
		return nil, err
	}

	return coreapi.NewCoreAPI(node)
}

// Spawns a node on the default repo location, if the repo exists
func spawnDefault(ctx context.Context) (iCore.CoreAPI, error) {
	defaultPath, err := config.PathRoot()
	if err != nil {
		// shouldn't be possible
		return nil, err
	}

	if err := setupPlugins(defaultPath); err != nil {
		return nil, err

	}

	return createNode(ctx, defaultPath)
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func spawnEphemeral(ctx context.Context) (iCore.CoreAPI, error) {
	// Create a Temporary Repo
	repoPath, err := createTempRepo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	if err := setupPlugins(repoPath); err != nil {
		return nil, err
	}

	// Spawning an ephemeral IPFS node
	return createNode(ctx, repoPath)
}

/// -------

func main() {
	fmt.Println("Getting an IPFS node running")

	ctx, _ := context.WithCancel(context.Background())

	/*
		fmt.Println("Spawning node on default repo")
		ipfs, err := spawnDefault(ctx)
		if err != nil {
			fmt.Println("No IPFS repo available on the default path")
		}
	*/

	fmt.Println("Spawning node on a temporary repo")
	ipfs, err := spawnEphemeral(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to spawn ephemeral node: %s", err))
	}

	fmt.Println("IPFS node running")

	testCIDStr := "QmUaoioqU7bxezBQZkUcgcSyokatMY71sxsALxQmRRrHrj"
	outputPath := "/Users/imp/Downloads/test-101/" + testCIDStr
	testCID := iCorePath.New(testCIDStr)

	out, err := ipfs.Unixfs().Get(ctx, testCID)
	if err != nil {
		panic(fmt.Errorf("Could not get CID: %s", err))
	}

	err = files.WriteTo(out, outputPath)
	if err != nil {
		panic(fmt.Errorf("Could not write out the fetched CID: %s", err))
	}
}
