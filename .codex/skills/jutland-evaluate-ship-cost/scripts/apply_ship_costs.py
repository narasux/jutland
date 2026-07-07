import json5, json, math, os, re

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))))
with open('/tmp/weapon_costs.json') as f:
    wp_costs = json.load(f)
with open(f'{REPO}/configs/ships.json5') as f:
    content = f.read()
    ships = json5.loads(content)

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
WEAPON_SCALE = 10
AIR_FIT_FACTOR = 0.12
AIR_TYPE_PENALTY = 1.5
AIR_FIT_MAX = 18

_plane_costs = {}
_plane_times = {}
with open(f'{REPO}/configs/planes.json5') as f:
    for p in json5.loads(f.read()):
        _plane_costs[p['name']] = p.get('fundsCost', 10)
        _plane_times[p['name']] = p.get('timeCost', 6)

def air_wing_fit_penalty(s):
    count = 0
    weighted_time = 0
    types = set()
    for g in s.get('aircraft',{}).get('groups',[]):
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

def calc(s):
    hp = float(s.get('totalHP', 0))
    stype = s.get('type', '?')
    nation = s.get('nation', 'jp')
    tm = TYPE_M.get(stype, 0.35)
    nm = NATION_M.get(nation, 1.0)
    hull_raw = math.pow(max(hp, 1), 0.45) * tm * nm * HULL_SF
    hull = max(0, min(1000, int(round(hull_raw / 5) * 5)))
    
    wcost = 0
    for field in ['mainGuns','secondaryGuns','antiAircraftGuns']:
        for w in s.get('weapon',{}).get(field,[]):
            wcost += wp_costs['guns'].get(w['name'],0)
    for w in s.get('weapon',{}).get('torpedoes',[]):
        wcost += wp_costs['torpedoes'].get(w['name'],0)
    for w in s.get('weapon',{}).get('rockets',[]):
        wcost += wp_costs['rockets'].get(w['name'],0)
    wcost //= WEAPON_SCALE
    
    # fundsCost = hull + weapons (no aircraft — those are runtime)
    funds = hull + wcost
    
    acost = 0
    for g in s.get('aircraft',{}).get('groups',[]):
        acost += g.get('maxCount',0) * _plane_costs.get(g.get('name',''), 10)
    
    total = funds + acost
    hull_time = max(3, min(130, int(round(funds * 0.35 + 2))))
    fit_penalty = air_wing_fit_penalty(s)
    time_cost = max(3, min(130, int(round(hull_time + fit_penalty * TIME_M.get(nation, 1.0)))))
    return funds, total, time_cost

cost_map = {}
for s in ships:
    funds, total, time_cost = calc(s)
    cost_map[s['name']] = (funds, total, time_cost)

lines = content.split('\n')
result = []
for i, line in enumerate(lines):
    if line.strip() in ('// TODO 确认资金', '// TODO 确认时间', '// TODO', '// FIXME 费用待定'):
        continue
    mf = re.match(r'^(\s{4})fundsCost:\s*\d+(\s*,?\s*)$', line)
    mt = re.match(r'^(\s{4})timeCost:\s*\d+(\s*,?\s*)$', line)
    if mf:
        for j in range(i-1, -1, -1):
            nm = re.match(r'^\s{4}name:\s*"([^"]+)"', lines[j])
            if nm and nm.group(1) in cost_map:
                line = f'{mf.group(1)}fundsCost: {cost_map[nm.group(1)][0]}{mf.group(2)}'
                break
    elif mt:
        for j in range(i-1, -1, -1):
            nm = re.match(r'^\s{4}name:\s*"([^"]+)"', lines[j])
            if nm and nm.group(1) in cost_map:
                line = f'{mt.group(1)}timeCost: {cost_map[nm.group(1)][2]}{mt.group(2)}'
                break
    result.append(line)

with open(f'{REPO}/configs/ships.json5', 'w') as f:
    f.write('\n'.join(result) + '\n')

print(f'Updated {len(cost_map)} ships')
for name in ['T_14','PT_791','ardent','atlanta','baltimore','warspite','yamato','hosho','saratoga','shinano']:
    if name in cost_map:
        f_, t, tm = cost_map[name]
        ac = t - f_
        print(f'  {name:20s}: funds=${f_:>4d} (+air={ac:>5d}$ ={t:>6d}$) time={tm:>3d}s')
