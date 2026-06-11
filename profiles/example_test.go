package profiles_test

import (
	"context"
	"fmt"
	"os"

	"github.com/dash0hq/dash0-api-client-go/profiles"
)

func ExampleNewStore() {
	svc, err := profiles.NewStore()
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	allProfiles, err := svc.GetProfiles()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("found %d profiles\n", len(allProfiles))
}

func ExampleNewStore_withConfigDir() {
	svc, err := profiles.NewStore(profiles.WithConfigDir("/tmp/dash0-test"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	allProfiles, err := svc.GetProfiles()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("found %d profiles\n", len(allProfiles))

	// Output:
	// found 0 profiles
}

func ExampleWithConfiguration() {
	cfg := &profiles.Configuration{
		ApiUrl:    "https://api.eu-west-1.aws.dash0.com",
		AuthToken: "auth_example-token",
		Dataset:   "production",
	}

	ctx := profiles.WithConfiguration(context.Background(), cfg)

	retrieved := profiles.FromContext(ctx)
	fmt.Println(retrieved.ApiUrl)

	// Output:
	// https://api.eu-west-1.aws.dash0.com
}

func ExampleStore_GetActiveConfiguration() {
	// Create a temporary config directory with a profile.
	configDir, _ := os.MkdirTemp("", "dash0-example-*")
	defer func() { _ = os.RemoveAll(configDir) }()

	store, _ := profiles.NewStore(profiles.WithConfigDir(configDir))
	_ = store.AddProfile(profiles.Profile{
		Name: "dev",
		Configuration: profiles.Configuration{
			ApiUrl:    "https://api.eu-west-1.aws.dash0.com",
			AuthToken: "auth_dev-token",
			Dataset:   "staging",
		},
	})

	cfg, err := store.GetActiveConfiguration()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cfg.ApiUrl)
	fmt.Println(cfg.Dataset)

	// Output:
	// https://api.eu-west-1.aws.dash0.com
	// staging
}

func ExampleStore_GetActiveConfiguration_envVarOverride() {
	// Environment variables override values from the active profile.
	configDir, _ := os.MkdirTemp("", "dash0-example-*")
	defer func() { _ = os.RemoveAll(configDir) }()

	store, _ := profiles.NewStore(profiles.WithConfigDir(configDir))
	_ = store.AddProfile(profiles.Profile{
		Name: "dev",
		Configuration: profiles.Configuration{
			ApiUrl:    "https://api.eu-west-1.aws.dash0.com",
			AuthToken: "auth_dev-token",
			Dataset:   "staging",
		},
	})

	// DASH0_DATASET overrides the profile's dataset.
	_ = os.Setenv(profiles.EnvDataset, "production")
	defer func() { _ = os.Unsetenv(profiles.EnvDataset) }()

	cfg, err := store.GetActiveConfiguration()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cfg.ApiUrl)
	fmt.Println(cfg.Dataset)

	// Output:
	// https://api.eu-west-1.aws.dash0.com
	// production
}

func ExampleResolveConfiguration_parameterOverride() {
	// Parameters take highest precedence, overriding both profiles and env vars.
	configDir, _ := os.MkdirTemp("", "dash0-example-*")
	defer func() { _ = os.RemoveAll(configDir) }()

	store, _ := profiles.NewStore(profiles.WithConfigDir(configDir))
	_ = store.AddProfile(profiles.Profile{
		Name: "dev",
		Configuration: profiles.Configuration{
			ApiUrl:    "https://api.eu-west-1.aws.dash0.com",
			AuthToken: "auth_dev-token",
		},
	})

	cfg, err := profiles.ResolveConfiguration(
		"https://api.us-west-2.aws.dash0.com", // overrides profile's API URL
		"",                                    // falls back to profile's auth token
		profiles.WithConfigDir(configDir),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cfg.ApiUrl)
	fmt.Println(cfg.AuthToken)

	// Output:
	// https://api.us-west-2.aws.dash0.com
	// auth_dev-token
}

func ExampleConfiguration_DatasetPtr() {
	cfg := &profiles.Configuration{Dataset: "production"}
	ptr := cfg.DatasetPtr()
	fmt.Println(*ptr)

	cfgDefault := &profiles.Configuration{Dataset: "default"}
	ptrDefault := cfgDefault.DatasetPtr()
	fmt.Println(ptrDefault)

	// Output:
	// production
	// <nil>
}

func ExampleConfiguration_ClientOptions() {
	cfg := &profiles.Configuration{
		ApiUrl:    "https://api.eu-west-1.aws.dash0.com",
		AuthToken: "auth_example-token",
	}

	opts := cfg.ClientOptions()
	fmt.Printf("produced %d client options\n", len(opts))

	// Output:
	// produced 2 client options
}

func ExampleNewOAuthClientStore() {
	store, err := profiles.NewOAuthClientStore(profiles.WithConfigDir("/tmp/dash0-example-clients"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	_, found, err := store.Get("https://api.eu-west-1.aws.dash0.com")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("found:", found)
	// Output:
	// found: false
}

func ExampleCanonicalAPIURL() {
	key, err := profiles.CanonicalAPIURL("HTTPS://API.EU-WEST-1.AWS.DASH0.COM/?x=1#frag")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(key)
	// Output: https://api.eu-west-1.aws.dash0.com
}

func ExampleOAuthClientStore_Put() {
	configDir, _ := os.MkdirTemp("", "dash0-example-*")
	defer func() { _ = os.RemoveAll(configDir) }()

	store, err := profiles.NewOAuthClientStore(profiles.WithConfigDir(configDir))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := store.Put("https://api.eu-west-1.aws.dash0.com", profiles.OAuthClientRecord{
		ClientID:    "client-123",
		RedirectURI: "http://localhost:8080/callback",
	}); err != nil {
		fmt.Println("error:", err)
		return
	}
	rec, found, _ := store.Get("https://api.eu-west-1.aws.dash0.com")
	fmt.Println(found, rec.ClientID)
	// Output: true client-123
}
