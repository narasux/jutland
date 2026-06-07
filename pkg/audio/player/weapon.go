package player

import (
	baseAudio "github.com/narasux/jutland/pkg/audio"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
)

const (
	rocketSpawnAudioCooldownTick   = 18
	rocketExplodeAudioCooldownTick = 12
	rocketSpawnVolume              = 0.10
	rocketExplodeVolume            = 0.08
)

// WeaponFire 统一管理武器发射相关短音效，并对高频音效做冷却去重。
type WeaponFire struct {
	rocketSpawnAudioCD   int
	rocketExplodeAudioCD int
}

// NewWeaponFire 创建武器发射音效播放器。
func NewWeaponFire() *WeaponFire {
	return &WeaponFire{}
}

// Update 推进音效冷却计数，应每帧调用一次。
func (p *WeaponFire) Update() {
	if p.rocketSpawnAudioCD > 0 {
		p.rocketSpawnAudioCD--
	}
	if p.rocketExplodeAudioCD > 0 {
		p.rocketExplodeAudioCD--
	}
}

// PlayShipFire 按战舰本帧发射事件播放音效，炮声优先按最大口径结算。
func (p *WeaponFire) PlayShipFire(maxBulletDiameter int, torpedoLaunched, rocketLaunched bool) {
	if maxBulletDiameter > 0 {
		baseAudio.PlayAudioToEnd(audioRes.NewGunFire(maxBulletDiameter))
	}
	if rocketLaunched {
		p.PlayRocketSpawn()
	} else if maxBulletDiameter == 0 && torpedoLaunched {
		baseAudio.PlayAudioToEnd(audioRes.NewTorpedoLaunch())
	}
}

// PlayPlaneFire 按战机本帧投弹或鱼雷发射事件播放音效。
func (p *WeaponFire) PlayPlaneFire(bombReleased, torpedoLaunched bool) {
	if bombReleased {
		baseAudio.PlayAudioToEnd(audioRes.NewBombSpawn())
	} else if torpedoLaunched {
		baseAudio.PlayAudioToEnd(audioRes.NewTorpedoLaunch())
	}
}

// PlayRocketSpawn 播放受冷却控制的火箭发射音，避免密集连发时声音堆叠。
func (p *WeaponFire) PlayRocketSpawn() {
	if p.rocketSpawnAudioCD > 0 {
		return
	}
	baseAudio.PlayAudioToEndWithVolume(audioRes.NewRocketSpawn(), rocketSpawnVolume)
	p.rocketSpawnAudioCD = rocketSpawnAudioCooldownTick
}

// PlayRocketExplode 播放受冷却控制的火箭爆炸音，避免空爆密集时声音堆叠。
func (p *WeaponFire) PlayRocketExplode() {
	if p.rocketExplodeAudioCD > 0 {
		return
	}
	baseAudio.PlayAudioToEndWithVolume(audioRes.NewRocketExplode(), rocketExplodeVolume)
	p.rocketExplodeAudioCD = rocketExplodeAudioCooldownTick
}
