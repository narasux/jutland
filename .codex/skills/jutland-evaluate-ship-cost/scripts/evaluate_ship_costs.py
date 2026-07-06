#!/usr/bin/env python3
"""Compute ship costs v2: HP^0.45 hull + scaled weapons + aircraft."""

import json5, json, math

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
NATION_M = {'us': 0.90, 'jp': 1.00, 'de': 1.05, 'uk': 1.00, 'ru': 1.10, 'cn': 1.00, 'special': 0.00}
HULL_SF = 3.6
WEAPON_SCALE = 10  # divide raw weapon costs

_plane_costs = {}
with open(f'{REPO}/configs/planes.json5') as f:
    for p in json5.loads(f.read()):
        _plane_costs[p['name']] = p.get('fundsCost', 10)

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
    time_cost = int(round(total * 0.35 + 2))
    time_cost = max(3, min(130, time_cost))

    records.append((name, stype, nation, hp, hull, wcost, acost, total, time_cost))

records.sort(key=lambda r: r[4]+r[5]+r[6])  # sort by total

print(f"{'name':20s} {'type':16s} {'nat':4s} {'HP':>8s} {'hull':>5s} {'weap':>4s} {'air':>6s} {'total':>6s} {'time':>5s}")
print('-'*85)
for r in records:
    name, stype, nation, hp, hull, wcost, acost, total, time_cost = r
    print(f'{name:20s} {stype:16s} {nation:4s} {hp:>8.0f} {hull:>4d}$ {wcost:>4d}$ {acost:>5d}$ {total:>5d}$ {time_cost:>4d}s')

buckets = {}
for r in records: buckets[r[7]] = buckets.get(r[7],0)+1
print(f'\nTotal dist: {dict(sorted(buckets.items()))}')
