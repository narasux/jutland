package cheat

// Cheats 秘籍表
var Cheats = []Cheat{
	// 对象类
	&ShowMeTheDuck{},
	&ShowMeTheWaterdrop{},
	&ShowMeTheMolaMola{},
	&BlackGoldRush{},
	// 经济类
	&ShowMeTheMoney{},
	// 效果类
	&AngelicaSinensis{},
	&BlackSheepWall{},
	&BathtubWar{},
	&WhoIsCallingTheFleet{},
	&DoNotDie{},
	&YouHaveBetrayedTheWorkingClass{},
	&AbandonDarkness{},
	&Expelliarmus{},
}

// DebugCheats 调试秘籍表
var DebugCheats = []Cheat{
	&DebugAll{},
	&DamageColorByTeam{},
	&ShowCursorPosObjInfo{},
}
