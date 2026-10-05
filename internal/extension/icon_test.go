package extension

import (
	"encoding/json"
	"encoding/xml"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shopware/shopware-cli/internal/testhelper"
)

func TestResizeExtensionIcon(t *testing.T) {
	t.Run("no icon path", func(t *testing.T) {
		ext := &mockExtension{iconPath: ""}
		err := ResizeExtensionIcon(getTestContext(), ext)
		assert.NoError(t, err)
	})

	t.Run("icon path is set but file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		iconPath := filepath.Join(tempDir, "missing.png")

		ext := &mockExtension{iconPath: iconPath}
		err := ResizeExtensionIcon(getTestContext(), ext)
		assert.NoError(t, err)
	})

	t.Run("icon is smaller than 56x56", func(t *testing.T) {
		tempDir := t.TempDir()
		iconPath := filepath.Join(tempDir, "icon.png")
		assert.NoError(t, createTestImageWithSize(iconPath, 32, 32))

		ext := &mockExtension{iconPath: iconPath}
		err := ResizeExtensionIcon(getTestContext(), ext)
		assert.NoError(t, err)

		f, err := os.Open(iconPath)
		assert.NoError(t, err)

		img, _, err := image.DecodeConfig(f)
		assert.NoError(t, err)
		assert.Equal(t, 32, img.Width)
		assert.Equal(t, 32, img.Height)
		assert.NoError(t, f.Close(), "Failed to close file after resizing icon")
	})

	t.Run("icon is exactly 56x56", func(t *testing.T) {
		tempDir := t.TempDir()
		iconPath := filepath.Join(tempDir, "icon.png")
		assert.NoError(t, createTestImageWithSize(iconPath, 32, 32))

		ext := &mockExtension{iconPath: iconPath}
		err := ResizeExtensionIcon(getTestContext(), ext)
		assert.NoError(t, err)

		f, err := os.Open(iconPath)
		assert.NoError(t, err)

		img, _, err := image.DecodeConfig(f)
		assert.NoError(t, err)
		assert.Equal(t, 32, img.Width)
		assert.Equal(t, 32, img.Height)
		assert.NoError(t, f.Close(), "Failed to close file after resizing icon")
	})

	t.Run("icon is larger than 56x56", func(t *testing.T) {
		tempDir := t.TempDir()
		iconPath := filepath.Join(tempDir, "icon.png")
		assert.NoError(t, createTestImageWithSize(iconPath, 32, 32))

		ext := &mockExtension{iconPath: iconPath}
		err := ResizeExtensionIcon(getTestContext(), ext)
		assert.NoError(t, err)

		f, err := os.Open(iconPath)
		assert.NoError(t, err)

		img, _, err := image.DecodeConfig(f)
		assert.NoError(t, err)
		assert.Equal(t, 32, img.Width)
		assert.Equal(t, 32, img.Height)
		assert.NoError(t, f.Close(), "Failed to close file after resizing icon")
	})
}

func TestRemoveExtensionIcon(t *testing.T) {
	t.Run("plugin with an icon reference", func(t *testing.T) {
		composer := testhelper.PluginComposer("frosh/frosh-test", "1.0.0", `FroshTest\FroshTest`)
		composer.Extra = map[string]any{"plugin-icon": "src/Resources/config/plugin.png"}

		dir := testhelper.ExtensionDir(t, composer)
		iconPath := filepath.Join(dir, "src", "Resources", "config", "plugin.png")

		require.NoError(t, os.MkdirAll(filepath.Dir(iconPath), 0o755))
		require.NoError(t, createTestImage(iconPath))

		ext, err := GetExtensionByFolder(getTestContext(), dir)
		require.NoError(t, err)
		require.Equal(t, iconPath, ext.GetIconPath())

		require.NoError(t, RemoveExtensionIcon(getTestContext(), ext))

		assert.NoFileExists(t, iconPath)

		composerJson, err := os.ReadFile(filepath.Join(dir, "composer.json"))
		require.NoError(t, err)

		var parsed struct {
			Name  string `json:"name"`
			Extra struct {
				PluginClass string `json:"shopware-plugin-class"`
				Label       any    `json:"label"`
			} `json:"extra"`
		}

		require.NoError(t, json.Unmarshal(composerJson, &parsed))
		assert.Equal(t, "frosh/frosh-test", parsed.Name)
		assert.Equal(t, `FroshTest\FroshTest`, parsed.Extra.PluginClass)
		assert.NotEmpty(t, parsed.Extra.Label)
		assert.NotContains(t, string(composerJson), "plugin-icon")
	})

	t.Run("plugin without an icon", func(t *testing.T) {
		dir := testhelper.ExtensionDir(t, testhelper.PluginComposer("frosh/frosh-test", "1.0.0", `FroshTest\FroshTest`))
		composerPath := filepath.Join(dir, "composer.json")

		before, err := os.ReadFile(composerPath)
		require.NoError(t, err)

		ext, err := GetExtensionByFolder(getTestContext(), dir)
		require.NoError(t, err)

		require.NoError(t, RemoveExtensionIcon(getTestContext(), ext))

		after, err := os.ReadFile(composerPath)
		require.NoError(t, err)
		assert.Equal(t, string(before), string(after))
	})

	t.Run("app with an icon reference", func(t *testing.T) {
		manifest := testhelper.NewAppManifest("MyExampleApp")
		manifest.Icon = "Resources/config/plugin.png"

		dir := testhelper.AppDir(t, manifest)
		iconPath := filepath.Join(dir, "Resources", "config", "plugin.png")

		require.NoError(t, os.MkdirAll(filepath.Dir(iconPath), 0o755))
		require.NoError(t, createTestImage(iconPath))

		ext, err := GetExtensionByFolder(getTestContext(), dir)
		require.NoError(t, err)
		require.Equal(t, iconPath, ext.GetIconPath())

		require.NoError(t, RemoveExtensionIcon(getTestContext(), ext))

		assert.NoFileExists(t, iconPath)

		newManifest, err := os.ReadFile(filepath.Join(dir, "manifest.xml"))
		require.NoError(t, err)
		assert.NotContains(t, string(newManifest), "<icon>")
		assert.Contains(t, string(newManifest), "<version>1.0.0</version>")
		assert.True(t, strings.HasPrefix(string(newManifest), xml.Header), "manifest keeps its XML declaration")
	})

	t.Run("app without an icon", func(t *testing.T) {
		dir := testhelper.AppDir(t, testhelper.NewAppManifest("MyExampleApp"))
		manifestPath := filepath.Join(dir, "manifest.xml")

		before, err := os.ReadFile(manifestPath)
		require.NoError(t, err)

		ext, err := GetExtensionByFolder(getTestContext(), dir)
		require.NoError(t, err)

		require.NoError(t, RemoveExtensionIcon(getTestContext(), ext))

		after, err := os.ReadFile(manifestPath)
		require.NoError(t, err)
		assert.Equal(t, string(before), string(after))
	})

	t.Run("extension type without an icon", func(t *testing.T) {
		ext := &mockExtension{iconPath: ""}

		assert.NoError(t, RemoveExtensionIcon(getTestContext(), ext))
	})
}
