package unit

import (
	"math"
	"testing"
)

func TestShipWeaponReloadStatusUsesNextReadyMount(t *testing.T) {
	now := int64(10_000)
	weapon := ShipWeapon{
		MainGunDisabled: true,
		MainGuns: []*Gun{
			{ReloadStartAt: 0, ReloadTime: 1},
			{ReloadStartAt: 9_000, ReloadTime: 2},
			{ReloadStartAt: 8_000, ReloadTime: 4},
		},
	}

	status := weapon.ReloadStatus(WeaponTypeMainGun, now)
	if status.Equipped != 3 || status.Ready != 1 {
		t.Fatalf("equipment status = %+v, want 1/3 ready", status)
	}
	if status.RemainingMillis != 1_000 {
		t.Fatalf("remaining = %d, want 1000", status.RemainingMillis)
	}
	if math.Abs(status.Progress-0.5) > 1e-9 {
		t.Fatalf("progress = %v, want 0.5", status.Progress)
	}
	if !status.Disabled {
		t.Fatal("disabled weapon category reported as enabled")
	}
}

func TestTorpedoReloadStatusAccountsForShotIntervalAndReload(t *testing.T) {
	now := int64(10_000)
	weapon := ShipWeapon{Torpedoes: []*TorpedoLauncher{
		{
			ReloadStartAt: 0,
			ReloadTime:    2,
			LatestFireAt:  9_500,
			ShotInterval:  1,
			BulletCount:   4,
		},
		{
			ReloadStartAt: 9_000,
			ReloadTime:    5,
			LatestFireAt:  0,
			ShotInterval:  1,
			BulletCount:   4,
		},
	}}

	status := weapon.ReloadStatus(WeaponTypeTorpedo, now)
	if status.Ready != 0 || status.RemainingMillis != 500 {
		t.Fatalf("status = %+v, want next launcher in 500ms", status)
	}
	if math.Abs(status.Progress-0.5) > 1e-9 {
		t.Fatalf("progress = %v, want 0.5", status.Progress)
	}
}

func TestRocketReloadStatusUsesGroupInterval(t *testing.T) {
	weapon := ShipWeapon{Rockets: []*RocketLauncher{{
		RocketCount:           6,
		GroupCount:            3,
		ShotInterval:          0.5,
		GroupInterval:         4,
		ReloadTime:            8,
		ReloadStartAt:         0,
		LatestFireAt:          9_000,
		ShotCountBeforeReload: 2,
	}}}
	status := weapon.ReloadStatus(WeaponTypeRocket, 10_000)
	if status.RemainingMillis != 3_000 {
		t.Fatalf("remaining = %d, want 3000", status.RemainingMillis)
	}
	if math.Abs(status.Progress-0.25) > 1e-9 {
		t.Fatalf("progress = %v, want 0.25", status.Progress)
	}
}

func TestReloadStatusAllReady(t *testing.T) {
	weapon := ShipWeapon{SecondaryGuns: []*Gun{{ReloadStartAt: 0, ReloadTime: 1}}}
	status := weapon.ReloadStatus(WeaponTypeSecondaryGun, 10_000)
	if status.Ready != 1 || status.Progress != 1 || status.RemainingMillis != 0 {
		t.Fatalf("status = %+v, want fully ready", status)
	}
}
