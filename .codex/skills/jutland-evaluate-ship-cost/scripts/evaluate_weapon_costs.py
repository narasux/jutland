#!/usr/bin/env python3
"""Compute weapon costs: guns, torpedo launchers, rocket launchers."""

import json5, math, sys, os

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))))

# Load bullets for damage lookup
with open(f'{REPO}/configs/bullets.json5') as f:
    bullets = {b['name']: b for b in json5.loads(f.read())}

def bullet_damage(bullet_name):
    b = bullets.get(bullet_name, {})
    return float(b.get('damage', 0))

def compute_cost(bullet_name, bullet_count, reload_time, type_factor):
    dmg = bullet_damage(bullet_name)
    if dmg <= 0 or bullet_count <= 0 or reload_time <= 0:
        return 1
    raw = dmg * bullet_count / reload_time * type_factor
    cost = int(round(raw / 5) * 5)
    return max(1, min(200, cost))

# Process guns
with open(f'{REPO}/configs/guns.json5') as f:
    guns = json5.loads(f.read())

gun_costs = {}
for g in guns:
    cost = compute_cost(g['bulletName'], g.get('bulletCount', 1), g.get('reloadTime', 1), 0.5)
    gun_costs[g['name']] = cost
    print(f"GUN  {g['name']:30s} dmg={bullet_damage(g['bulletName']):6.0f} ct={g.get('bulletCount',1):2d} rt={g.get('reloadTime',1):5.1f}s  cost=${cost:3d}")

print()

# Process torpedo launchers
with open(f'{REPO}/configs/torpedo_launchers.json5') as f:
    torps = json5.loads(f.read())

torp_costs = {}
for t in torps:
    # For torpedoes, use the full cycle time: reloadTime + (count-1)*shotInterval
    cycle = t.get('reloadTime', 1) + max(0, t.get('bulletCount', 1) - 1) * t.get('shotInterval', 0)
    cost = compute_cost(t['bulletName'], t.get('bulletCount', 1), cycle, 1.0)
    torp_costs[t['name']] = cost
    print(f"TORP {t['name']:30s} dmg={bullet_damage(t['bulletName']):6.0f} ct={t.get('bulletCount',1):2d} cyc={cycle:5.1f}s  cost=${cost:3d}")

print()

# Process rocket launchers
with open(f'{REPO}/configs/rocket_launchers.json5') as f:
    rockets = json5.loads(f.read())

rocket_costs = {}
for r in rockets:
    # Same cycle calculation as torpedo
    cycle = r.get('reloadTime', 1) + max(0, r.get('rocketCount', 1) - 1) * r.get('shotInterval', 0)
    cost = compute_cost(r['bulletName'], r.get('rocketCount', 1), cycle, 0.7)
    rocket_costs[r['name']] = cost
    print(f"RKT  {r['name']:30s} dmg={bullet_damage(r['bulletName']):6.0f} ct={r.get('rocketCount',1):2d} cyc={cycle:5.1f}s  cost=${cost:3d}")

# Save costs to a file for the ship cost script
import json
with open('/tmp/weapon_costs.json', 'w') as f:
    json.dump({'guns': gun_costs, 'torpedoes': torp_costs, 'rockets': rocket_costs}, f, indent=2)
print(f"\nSaved {len(gun_costs)} guns + {len(torp_costs)} torpedoes + {len(rocket_costs)} rockets to /tmp/weapon_costs.json")
