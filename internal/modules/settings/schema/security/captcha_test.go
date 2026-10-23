package settingssecurity

import (
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
)

func boolPointer(value bool) *bool { return &value }

func TestCaptchaRegisterSceneRoundTrip(t *testing.T) {
	cfg := config.CaptchaConfig{
		Provider: constants.CaptchaProviderImage,
		Scenes:   config.CaptchaSceneConfig{Register: true},
	}
	setting := DefaultCaptchaSetting(cfg)
	if !setting.Scenes.Register {
		t.Fatal("register scene must be enabled from config")
	}
	if !setting.IsSceneEnabled(constants.CaptchaSceneRegister) {
		t.Fatal("IsSceneEnabled(register) should be true")
	}

	encoded := EncodeCaptchaSetting(setting)
	scenes, _ := encoded["scenes"].(map[string]interface{})
	if scenes["register"] != true {
		t.Fatalf("encoded register scene missing: %#v", encoded["scenes"])
	}

	decoded := DecodeCaptchaSetting(encoded, DefaultCaptchaSetting(config.CaptchaConfig{Provider: constants.CaptchaProviderImage}))
	if !decoded.Scenes.Register {
		t.Fatal("register scene lost after decode")
	}

	public := PublicCaptchaSetting(setting)
	publicScenes, _ := public["scenes"].(map[string]interface{})
	if publicScenes["register"] != true {
		t.Fatalf("public register scene missing: %#v", public["scenes"])
	}

	admin := MaskCaptchaSettingForAdmin(setting)
	adminScenes, _ := admin["scenes"].(map[string]interface{})
	if adminScenes["register"] != true {
		t.Fatalf("admin register scene missing: %#v", admin["scenes"])
	}

	runtime := CaptchaSettingToConfig(setting)
	if !runtime.Scenes.Register {
		t.Fatal("runtime config register scene missing")
	}

	patched, err := ApplyCaptchaSettingPatch(setting, CaptchaSettingPatch{
		Scenes: &CaptchaScenePatch{Register: boolPointer(false)},
	})
	if err != nil {
		t.Fatalf("patch captcha setting: %v", err)
	}
	if patched.Scenes.Register {
		t.Fatal("register scene should be disabled by patch")
	}
}
