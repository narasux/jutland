#!/usr/bin/env python3
"""Compute ship costs v2: HP^0.45 hull + scaled weapons + aircraft."""

import json5, json, math, os

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))))
with open('/tmp/weapon_costs.json') as f:
    wp_costs = json.load(f)
with open(f'{REPO}/configs/ships.json5') as f:
    ships = json5.loads(f.read())

TYPE_M = {
    'default': 0.00, 'torpedo_boat': 0.10, 'destroyer': 0.18, 'frigate': 0.15,
    'cruiser': 0.35, 'battleship': 0.80, 'aircraft_carrier': 0.65,
    'cargo': 0.30, 'hospital': 0.50,
}
NATION_M = {
    'us': 0.90, 'jp': 1.00, 'de': 1.05, 'uk': 1.00,
    'ru': 1.10, 'su': 1.10, 'cn': 1.00, 'special': 0.00,
}
TIME_M = {
    'us': 0.75, 'jp': 1.00, 'de': 1.00, 'uk': 0.90,
    'ru': 1.05, 'su': 1.05, 'cn': 0.50, 'special': 1.00,
}
HULL_SF = 3.6
WEAPON_SCALE = 10  # divide raw weapon costs
AIR_FIT_FACTOR = 0.12
AIR_TYPE_PENALTY = 1.5
AIR_FIT_MAX = 18

_plane_costs = {}
_plane_times = {}
with open(f'{REPO}/configs/planes.json5') as f:
    for p in json5.loads(f.read()):
        _plane_costs[p['name']] = p.get('fundsCost', 10)
        _plane_times[p['name']] = p.get('timeCost', 6)

def weapon_total(ship):
    t = 0
    for field in ['mainGuns','secondaryGuns','antiAircraftGuns']:
        for w in ship.get('weapon',{}).get(field,[]):
            t += wp_costs['guns'].get(w['name'],0)
    for w in ship.get('weapon',{}).get('torpedoes',[]):
        t += wp_costs['torpedoes'].get(w['name'],0)
    for w in ship.get('weapon',{}).get('rockets',[]):
        t += wp_costs['rockets'].get(w['name'],0)
    return t // WEAPON_SCALE

def aircraft_total(ship):
    t = 0
    for g in ship.get('aircraft',{}).get('groups',[]):
        t += g.get('maxCount',0) * _plane_costs.get(g.get('name',''), 10)
    return t

def air_wing_fit_penalty(ship):
    count = 0
    weighted_time = 0
    types = set()
    for g in ship.get('aircraft',{}).get('groups',[]):
        plane_name = g.get('name','')
        plane_count = g.get('maxCount',0)
        if plane_count <= 0:
            continue
        count += plane_count
        weighted_time += plane_count * _plane_times.get(plane_name, 6)
        types.add(plane_name)
    if count <= 0:
        return 0
    avg_time = weighted_time / count
    raw = math.sqrt(count) * avg_time * AIR_FIT_FACTOR + max(0, len(types) - 1) * AIR_TYPE_PENALTY
    return max(0, min(AIR_FIT_MAX, int(round(raw))))

records = []
for s in ships:
    name = s['name']
    stype = s.get('type','?')
    nation = s.get('nation','jp')
    hp = float(s.get('totalHP',0))
    speed = float(s.get('maxSpeed',20))
    hdr = float(s.get('horizontalDamageReduction',0))
    vdr = float(s.get('verticalDamageReduction',0))
    tm = TYPE_M.get(stype, 0.35)
    nm = NATION_M.get(nation, 1.0)

    hull_raw = math.pow(max(hp,1), 0.45) * tm * nm * HULL_SF
    hull = int(round(hull_raw / 5) * 5)
    hull = max(0, min(1000, hull))

    wcost = weapon_total(s)
    acost = aircraft_total(s)
    total = hull + wcost + acost
    funds = hull + wcost
    hull_time = max(3, min(130, int(round(funds * 0.35 + 2))))
    fit_penalty = air_wing_fit_penalty(s)
    time_cost = int(round(hull_time + fit_penalty * TIME_M.get(nation, 1.0)))
    time_cost = max(3, min(130, time_cost))

    records.append((name, stype, nation, hp, hull, wcost, acost, total, fit_penalty, time_cost))

records.sort(key=lambda r: r[4]+r[5]+r[6])  # sort by total

print(f"{'name':20s} {'type':16s} {'nat':4s} {'HP':>8s} {'hull':>5s} {'weap':>4s} {'air':>6s} {'total':>6s} {'fit':>4s} {'time':>5s}")
print('-'*91)
for r in records:
    name, stype, nation, hp, hull, wcost, acost, total, fit_penalty, time_cost = r
    print(f'{name:20s} {stype:16s} {nation:4s} {hp:>8.0f} {hull:>4d}$ {wcost:>4d}$ {acost:>5d}$ {total:>5d}$ {fit_penalty:>3d}s {time_cost:>4d}s')

buckets = {}
for r in records: buckets[r[7]] = buckets.get(r[7],0)+1
print(f'\nTotal dist: {dict(sorted(buckets.items()))}')
