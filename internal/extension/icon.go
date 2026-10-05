package extension

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"

	"github.com/shopware/shopware-cli/internal/xmlpath"
	"github.com/shopware/shopware-cli/logging"
)

func ResizeExtensionIcon(ctx context.Context, ext Extension) error {
	// Resize the icon to 256x256 if it's larger than that.
	iconPath := ext.GetIconPath()
	if iconPath == "" {
		return nil // No icon to resize
	}

	file, err := os.Open(iconPath)
	if err != nil {
		if os.IsNotExist(err) {
			logging.FromContext(ctx).Infof("Extension icon %s does not exist, skipping resize", iconPath)
			return nil // No icon to resize
		}

		return fmt.Errorf("cannot open icon: %w", err)
	}

	src, _, err := image.Decode(file)
	if err := file.Close(); err != nil {
		return fmt.Errorf("cannot close icon file: %w", err)
	}

	if err != nil {
		return fmt.Errorf("cannot decode icon: %w", err)
	}

	if src.Bounds().Dx() <= 256 && src.Bounds().Dy() <= 256 {
		return nil
	}

	logging.FromContext(ctx).Infof("Resizing extension icon from %dx%d to 256x256", src.Bounds().Dx(), src.Bounds().Dy())

	dst := image.NewRGBA(image.Rect(0, 0, 256, 256))

	draw.BiLinear.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

	// Open file for writing
	writeFile, err := os.Create(iconPath)
	if err != nil {
		return fmt.Errorf("cannot create icon file: %w", err)
	}

	encoder := png.Encoder{
		CompressionLevel: png.BestCompression,
	}

	if err := encoder.Encode(writeFile, dst); err != nil {
		return fmt.Errorf("cannot encode icon: %w", err)
	}

	return writeFile.Close()
}

// RemoveExtensionIcon removes the icon of a prepared extension folder and
// clears every reference to it, so the package installs without an icon. A
// missing icon file and a missing reference are not an error.
func RemoveExtensionIcon(ctx context.Context, ext Extension) error {
	iconPath := ext.GetIconPath()
	fileRemoved := false

	if iconPath != "" {
		err := os.Remove(iconPath)
		switch {
		case err == nil:
			fileRemoved = true
		case !os.IsNotExist(err):
			return fmt.Errorf("cannot remove icon %s: %w", iconPath, err)
		}
	}

	referenceRemoved := false

	var err error

	switch ext.GetType() {
	case TypePlatformApp:
		referenceRemoved, err = removeAppIconReference(ext.GetPath())
	case TypePlatformPlugin:
		referenceRemoved, err = removePluginIconReference(ext.GetPath())
	}

	if err != nil {
		return err
	}

	if fileRemoved || referenceRemoved {
		logging.FromContext(ctx).Infof("Removed the extension icon from the package: %s", iconPath)
	}

	return nil
}

func removeAppIconReference(extensionRoot string) (bool, error) {
	manifestPath := filepath.Join(extensionRoot, "manifest.xml")

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return false, fmt.Errorf("cannot read manifest.xml: %w", err)
	}

	manifest, err := xmlpath.Parse(manifestBytes)
	if err != nil {
		return false, fmt.Errorf("cannot parse manifest.xml: %w", err)
	}

	if manifest.Root().RemoveAll("meta/icon") == 0 {
		return false, nil
	}

	newManifest, err := manifest.MarshalIndent("", "  ")
	if err != nil {
		return false, fmt.Errorf("cannot marshal manifest.xml: %w", err)
	}

	newManifest = append([]byte(xml.Header), newManifest...)

	if err := os.WriteFile(manifestPath, newManifest, 0o644); err != nil {
		return false, fmt.Errorf("cannot write manifest.xml: %w", err)
	}

	return true, nil
}

func removePluginIconReference(extensionRoot string) (bool, error) {
	composerJsonPath := filepath.Join(extensionRoot, "composer.json")

	composerJsonBytes, err := os.ReadFile(composerJsonPath)
	if err != nil {
		return false, fmt.Errorf("cannot read composer.json: %w", err)
	}

	var composerJsonStruct map[string]interface{}

	if err := json.Unmarshal(composerJsonBytes, &composerJsonStruct); err != nil {
		return false, fmt.Errorf("cannot parse composer.json: %w", err)
	}

	extra, ok := composerJsonStruct["extra"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	if _, ok := extra["plugin-icon"]; !ok {
		return false, nil
	}

	delete(extra, "plugin-icon")

	newComposerJson, err := json.MarshalIndent(composerJsonStruct, "", "  ")
	if err != nil {
		return false, fmt.Errorf("cannot marshal composer.json: %w", err)
	}

	if err := os.WriteFile(composerJsonPath, newComposerJson, 0o644); err != nil {
		return false, fmt.Errorf("cannot write composer.json: %w", err)
	}

	return true, nil
}
