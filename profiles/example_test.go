package profiles_test

import (
	"context"
	"fmt"
	"os"
	"time"

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
	configDir, _ := os.MkdirTemp("", "dash0-example-*")
	defer func() { _ = os.RemoveAll(configDir) }()

	svc, err := profiles.NewStore(profiles.WithConfigDir(configDir))
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
	configDir, _ := os.MkdirTemp("", "dash0-example-*")
	defer func() { _ = os.RemoveAll(configDir) }()

	store, err := profiles.NewOAuthClientStore(profiles.WithConfigDir(configDir))
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

func ExampleOAuthClientStore_Get() {
	configDir, _ := os.MkdirTemp("", "dash0-example-*")
	defer func() { _ = os.RemoveAll(configDir) }()

	store, _ := profiles.NewOAuthClientStore(profiles.WithConfigDir(configDir))
	_ = store.Put("https://api.eu-west-1.aws.dash0.com", profiles.OAuthClientRecord{
		ClientID:    "client-123",
		RedirectURI: "http://localhost:8080/callback",
	})

	rec, found, err := store.Get("https://api.eu-west-1.aws.dash0.com")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(found, rec.ClientID)
	// Output: true client-123
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

func ExampleOAuthClientStore_Delete() {
	configDir, _ := os.MkdirTemp("", "dash0-example-*")
	defer func() { _ = os.RemoveAll(configDir) }()

	store, _ := profiles.NewOAuthClientStore(profiles.WithConfigDir(configDir))
	_ = store.Put("https://api.eu-west-1.aws.dash0.com", profiles.OAuthClientRecord{
		ClientID:    "client-123",
		RedirectURI: "http://localhost:8080/callback",
	})

	if err := store.Delete("https://api.eu-west-1.aws.dash0.com"); err != nil {
		fmt.Println("error:", err)
		return
	}
	_, found, _ := store.Get("https://api.eu-west-1.aws.dash0.com")
	fmt.Println("found after delete:", found)
	// Output: found after delete: false
}

func ExampleStore_GetActiveConfigurationContext() {
	// The Context variant plumbs cancellation through the OAuth refresh
	// round-trip so a hung authorization server cannot pin the caller for
	// the full HTTP timeout.
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := store.GetActiveConfigurationContext(ctx)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cfg.ApiUrl)

	// Output:
	// https://api.eu-west-1.aws.dash0.com
}

func ExampleStore_AddProfileContext() {
	configDir, _ := os.MkdirTemp("", "dash0-example-*")
	defer func() { _ = os.RemoveAll(configDir) }()

	store, _ := profiles.NewStore(profiles.WithConfigDir(configDir))

	// The Context variant lets the caller bound the wait for the
	// cross-process .profile-lock acquisition — useful when a long-running
	// agent flow already has a deadline context to honor.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := store.AddProfileContext(ctx, profiles.Profile{
		Name: "dev",
		Configuration: profiles.Configuration{
			ApiUrl:    "https://api.eu-west-1.aws.dash0.com",
			AuthToken: "auth_dev-token",
		},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	all, _ := store.GetProfiles()
	fmt.Println("count:", len(all))
	// Output: count: 1
}

func ExampleStore_UpdateProfileContext() {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.UpdateProfileContext(ctx, "dev", func(cfg *profiles.Configuration) {
		cfg.Dataset = "production"
	}); err != nil {
		fmt.Println("error:", err)
		return
	}

	all, _ := store.GetProfiles()
	fmt.Println(all[0].Configuration.Dataset)
	// Output: production
}

func ExampleStore_SetActiveProfileContext() {
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
	_ = store.AddProfile(profiles.Profile{
		Name: "prod",
		Configuration: profiles.Configuration{
			ApiUrl:    "https://api.us-west-2.aws.dash0.com",
			AuthToken: "auth_prod-token",
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.SetActiveProfileContext(ctx, "prod"); err != nil {
		fmt.Println("error:", err)
		return
	}
	active, _ := store.GetActiveProfile()
	fmt.Println(active.Name)
	// Output: prod
}

func ExampleStore_RemoveProfile() {
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

	if err := store.RemoveProfile("dev"); err != nil {
		fmt.Println("error:", err)
		return
	}
	all, _ := store.GetProfiles()
	fmt.Println("remaining:", len(all))
	// Output: remaining: 0
}

func ExampleStore_RemoveProfileContext() {
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

	// The Context variant plumbs cancellation through the OAuth refresh-token
	// revocation HTTP call, so the caller can give up on a slow IdP.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.RemoveProfileContext(ctx, "dev"); err != nil {
		fmt.Println("error:", err)
		return
	}
	all, _ := store.GetProfiles()
	fmt.Println("remaining:", len(all))
	// Output: remaining: 0
}

func ExampleResolveConfigurationContext() {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := profiles.ResolveConfigurationContext(ctx, "", "",
		profiles.WithConfigDir(configDir),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cfg.ApiUrl)
	// Output: https://api.eu-west-1.aws.dash0.com
}

func ExampleResolveConfigurationWithOtlpContext() {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := profiles.ResolveConfigurationWithOtlpContext(ctx,
		"", "",
		"https://ingress.eu-west-1.aws.dash0.com",
		"production",
		profiles.WithConfigDir(configDir),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cfg.OtlpUrl, cfg.Dataset)
	// Output: https://ingress.eu-west-1.aws.dash0.com production
}
