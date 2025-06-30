package model

type Gender byte

const (
	GenderMale   Gender = 0
	GenderFemale Gender = 1
)

type ClassType byte

const (
	ClassTypeKnight  ClassType = 0
	ClassTypeWarrior ClassType = 1
	ClassTypeArcher  ClassType = 2
	ClassTypeMage    ClassType = 3
)

type SkinCarnation byte

const (
	SkinCarnationMaleBeige        SkinCarnation = 102
	SkinCarnationMalePink         SkinCarnation = 103
	SkinCarnationMaleLightBrown   SkinCarnation = 104
	SkinCarnationMaleBrown        SkinCarnation = 105
	SkinCarnationMaleGray         SkinCarnation = 106
	SkinCarnationFemaleBeige      SkinCarnation = 107
	SkinCarnationFemalePink       SkinCarnation = 108
	SkinCarnationFemaleLightBrown SkinCarnation = 109
	SkinCarnationFemaleBrown      SkinCarnation = 110
	SkinCarnationFemaleGray       SkinCarnation = 111
)

type HairStyle byte

const (
	HairStyleMaleShortWhite  HairStyle = 112
	HairStyleMaleShortBrown  HairStyle = 113
	HairStyleMaleShortBlack  HairStyle = 114
	HairStyleMaleShortRed    HairStyle = 115
	HairStyleMaleShortGray   HairStyle = 116
	HairStyleMaleMediumWhite HairStyle = 117
	HairStyleMaleMediumBrown HairStyle = 118
	HairStyleMaleMediumBlack HairStyle = 119
	HairStyleMaleMediumRed   HairStyle = 120
	HairStyleMaleMediumGray  HairStyle = 121
	HairStyleMaleLongWhite   HairStyle = 122
	HairStyleMaleLongBrown   HairStyle = 123
	HairStyleMaleLongBlack   HairStyle = 124
	HairStyleMaleLongRed     HairStyle = 125
	HairStyleMaleLongGray    HairStyle = 126
	HairStyleMaleBald        HairStyle = 200

	HairStyleFemaleShortWhite    HairStyle = 127
	HairStyleFemaleShortBrown    HairStyle = 129
	HairStyleFemaleShortBlack    HairStyle = 129
	HairStyleFemaleShortRed      HairStyle = 130
	HairStyleFemaleShortGray     HairStyle = 131
	HairStyleFemaleMediumWhite   HairStyle = 132
	HairStyleFemaleMediumBrown   HairStyle = 133
	HairStyleFemaleMediumBlack   HairStyle = 134
	HairStyleFemaleMediumRed     HairStyle = 135
	HairStyleFemaleMediumGray    HairStyle = 136
	HairStyleFemalePonytailWhite HairStyle = 137
	HairStyleFemalePonytailBrown HairStyle = 138
	HairStyleFemalePonytailBlack HairStyle = 139
	HairStyleFemalePonytailRed   HairStyle = 140
	HairStyleFemalePonytailGray  HairStyle = 141
	HairStyleFemaleLongWhite     HairStyle = 142
	HairStyleFemaleLongBrown     HairStyle = 143
	HairStyleFemaleLongBlack     HairStyle = 144
	HairStyleFemaleLongRed       HairStyle = 145
	HairStyleFemaleLongGray      HairStyle = 146
)
