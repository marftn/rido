package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marftn/rido/internal/fs"
)

func newIn(t *testing.T, contents string) (Config, error) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if contents != "" {
		dir := filepath.Join(home, ".config", "rido")
		if err := os.MkdirAll(dir, fs.FileModeDefault); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(contents), fs.FileModeReadOnly); err != nil {
			t.Fatal(err)
		}
	}

	return New()
}

func TestNew(t *testing.T) {
	for _, tc := range []struct {
		name, file, want string
		wantErr          bool
	}{
		{name: "no file", file: "", want: defaultStoreSubPath, wantErr: false},
		{name: "no key", file: `{}`, want: defaultStoreSubPath, wantErr: false},
		{name: "empty key", file: `{"storeRoot": ""}`, want: defaultStoreSubPath, wantErr: false},
		{name: "tilde", file: `{"storeRoot": "~/elsewhere"}`, want: "elsewhere", wantErr: false},
		{name: "bad json", file: `{`, want: "", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := newIn(t, tc.file)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			want := filepath.Join(os.Getenv("HOME"), tc.want)
			if cfg.StoreRoot != want {
				t.Errorf("StoreRoot = %q, want %q", cfg.StoreRoot, want)
			}
		})
	}
}

func TestNewAbsolutePath(t *testing.T) {
	cfg, err := newIn(t, `{"storeRoot": "/mnt/vault"}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StoreRoot != "/mnt/vault" {
		t.Errorf("StoreRoot = %q", cfg.StoreRoot)
	}
}
