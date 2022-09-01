package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func specInSlice(a proto.Spec, list []proto.Spec) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func writeGemFile(outDir string, gemsData []GemData) {
	err := os.MkdirAll(outDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(fmt.Sprintf("%s/all_gems.go", outDir))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString(`// DO NOT EDIT. This file is auto-generated by the item generator tool. Use that to make edits.
	
package items
	
import (
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

var Gems = []Gem{
`)

	for _, gemData := range gemsData {
		if gemData.Declaration.Filter {
			continue
		}
		allow := allowList[gemData.Declaration.ID]
		if !allow {
			if gemData.Response.GetQuality() < int(proto.ItemQuality_ItemQualityUncommon) {
				continue
			}
			// if gemData.Response.GetPhase() == 0 {
			// 	continue
			// }
		}
		file.WriteString(fmt.Sprintf("\t%s,\n", gemToGoString(gemData.Declaration, gemData.Response)))
	}

	file.WriteString("}\n")

	file.Sync()
}

func writeItemFile(outDir string, itemsData []ItemData) {
	err := os.MkdirAll(outDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(fmt.Sprintf("%s/all_items.go", outDir))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString(`// DO NOT EDIT. This file is auto-generated by the item generator tool. Use that to make edits.
	
package items
	
import (
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

var Items = []Item{
`)

	for _, itemData := range itemsData {
		if itemData.Response == nil {
			continue
		}
		if itemData.Declaration.Filter {
			continue
		}

		deny := false
		for _, pattern := range denyListNameRegexes {
			if pattern.MatchString(itemData.Response.GetName()) {
				deny = true
				break
			}
		}
		if deny {
			continue
		}

		if !itemData.Response.IsEquippable() {
			continue
		}

		itemLevel := itemData.Response.GetItemLevel()
		allow := allowList[itemData.Declaration.ID]
		if !allow {
			qual := itemData.Response.GetQuality()
			if qual < int(proto.ItemQuality_ItemQualityUncommon) {
				continue
			} else if qual > int(proto.ItemQuality_ItemQualityLegendary) {
				continue
			} else if qual < int(proto.ItemQuality_ItemQualityEpic) {
				if itemLevel < 145 {
					continue
				}
				if itemLevel < 149 && itemData.Response.GetItemSetName() == "" {
					continue
				}
			} else {
				// Epic and legendary items might come from classic, so use a lower ilvl threshold.
				if itemLevel < 140 {
					continue
				}
			}
		}
		if itemLevel == 0 {
			fmt.Printf("Missing ilvl: %s\n", itemData.Response.GetName())
		}

		file.WriteString(fmt.Sprintf("\t%s,\n", itemToGoString(itemData)))
	}

	file.WriteString("}\n")

	file.Sync()
}

func gemToGoString(gemDeclaration GemDeclaration, gemResponse ItemResponse) string {
	gemStr := "{"

	gemStr += fmt.Sprintf("Name:\"%s\", ", gemResponse.GetName())
	gemStr += fmt.Sprintf("ID:%d, ", gemDeclaration.ID)

	phase := gemDeclaration.Phase
	if phase == 0 {
		phase = gemResponse.GetPhase()
	}
	gemStr += fmt.Sprintf("Phase:%d, ", phase)
	gemStr += fmt.Sprintf("Quality:proto.ItemQuality_%s, ", proto.ItemQuality(gemResponse.GetQuality()).String())
	gemStr += fmt.Sprintf("Color:proto.GemColor_%s, ", proto.GemColor(gemResponse.GetSocketColor()).String())
	gemStr += fmt.Sprintf("Stats: %s, ", statsToGoString(gemResponse.GetGemStats(), gemDeclaration.Stats))

	if gemResponse.GetUnique() {
		gemStr += "Unique:true, "
	}

	prof := gemResponse.GetRequiredProfession()
	if prof != proto.Profession_ProfessionUnknown {
		gemStr += fmt.Sprintf("RequiredProfession:proto.Profession_%s, ", prof.String())
	}

	gemStr += "}"
	return gemStr
}

func itemToGoString(itemData ItemData) string {
	itemStr := "{"

	itemStr += fmt.Sprintf("Name:\"%s\", ", strings.ReplaceAll(itemData.Response.GetName(), "\"", "\\\""))
	itemStr += fmt.Sprintf("ID:%d, ", itemData.Declaration.ID)

	classAllowlist := itemData.Response.GetClassAllowlist()
	if len(itemData.Declaration.ClassAllowlist) > 0 {
		classAllowlist = itemData.Declaration.ClassAllowlist
	}
	if len(classAllowlist) > 0 {
		itemStr += "ClassAllowlist: []proto.Class{"
		for _, class := range classAllowlist {
			itemStr += fmt.Sprintf("proto.Class_%s,", class.String())
		}
		itemStr += "}, "
	}

	itemStr += fmt.Sprintf("Type:proto.ItemType_%s, ", itemData.Response.GetItemType().String())

	armorType := itemData.Response.GetArmorType()
	if armorType != proto.ArmorType_ArmorTypeUnknown {
		itemStr += fmt.Sprintf("ArmorType:proto.ArmorType_%s, ", armorType.String())
	}

	weaponType := itemData.Response.GetWeaponType()
	if weaponType != proto.WeaponType_WeaponTypeUnknown {
		itemStr += fmt.Sprintf("WeaponType:proto.WeaponType_%s, ", weaponType.String())

		handType := itemData.Response.GetHandType()
		if itemData.Declaration.HandType != proto.HandType_HandTypeUnknown {
			handType = itemData.Declaration.HandType
		}
		if handType == proto.HandType_HandTypeUnknown {
			panic("Unknown hand type for item: " + fmt.Sprintf("%#v", itemData.Response))
		}
		itemStr += fmt.Sprintf("HandType:proto.HandType_%s, ", handType.String())
	} else {
		rangedWeaponType := itemData.Response.GetRangedWeaponType()
		if rangedWeaponType != proto.RangedWeaponType_RangedWeaponTypeUnknown {
			itemStr += fmt.Sprintf("RangedWeaponType:proto.RangedWeaponType_%s, ", rangedWeaponType.String())
		}
	}

	min, max := itemData.Response.GetWeaponDamage()
	if min != 0 && max != 0 {
		itemStr += fmt.Sprintf("WeaponDamageMin: %0.1f, ", min)
		itemStr += fmt.Sprintf("WeaponDamageMax: %0.1f, ", max)
	}
	speed := itemData.Response.GetWeaponSpeed()
	if speed != 0 {
		itemStr += fmt.Sprintf("SwingSpeed: %0.2f, ", speed)
	}

	phase := itemData.Declaration.Phase
	if phase == 0 {
		phase = itemData.Response.GetPhase()
	}
	itemStr += fmt.Sprintf("Phase:%d, ", phase)
	itemStr += fmt.Sprintf("Quality:proto.ItemQuality_%s, ", proto.ItemQuality(itemData.Response.GetQuality()).String())

	if itemData.Response.GetUnique() {
		itemStr += "Unique:true, "
	}

	itemStr += fmt.Sprintf("Ilvl:%d, ", itemData.Response.GetItemLevel())

	if itemData.QualityModifier != 0 {
		itemStr += fmt.Sprintf("QualityModifier:%0.03f, ", itemData.QualityModifier)
	}

	itemStr += fmt.Sprintf("Stats: %s", statsToGoString(itemData.Response.GetStats(), itemData.Declaration.Stats))

	gemSockets := itemData.Response.GetGemSockets()
	if len(gemSockets) > 0 {
		itemStr += ", GemSockets: []proto.GemColor{"
		for _, gemColor := range gemSockets {
			itemStr += fmt.Sprintf("proto.GemColor_%s,", gemColor.String())
		}
		itemStr += "}, "
		itemStr += fmt.Sprintf("SocketBonus: %s", statsToGoString(itemData.Response.GetSocketBonus(), Stats{}))
	}

	setName := itemData.Response.GetItemSetName()
	if setName != "" {
		itemStr += fmt.Sprintf(", SetName: \"%s\"", setName)
	}

	prof := itemData.Response.GetRequiredProfession()
	if prof != proto.Profession_ProfessionUnknown {
		itemStr += fmt.Sprintf(", RequiredProfession:proto.Profession_%s", prof.String())
	}

	if itemData.Response.IsHeroic() {
		itemStr += ", Heroic: true"
	}

	itemStr += "}"
	return itemStr
}

func statsToGoString(statlist Stats, overrides Stats) string {
	statsStr := "stats.Stats{"

	for stat, value := range statlist {
		val := value
		if overrides[stat] > 0 {
			val = overrides[stat]
		}
		if value > 0 {
			statsStr += fmt.Sprintf("stats.%s:%.0f,", stats.Stat(stat).StatName(), val)
		}
	}

	statsStr += "}"
	return statsStr
}

// If any of these match the item name, don't include it.
var denyListNameRegexes = []*regexp.Regexp{
	regexp.MustCompile("PH\\]"),
	regexp.MustCompile("TEST"),
	regexp.MustCompile("Test"),
	regexp.MustCompile("Bracer 3"),
	regexp.MustCompile("Bracer 2"),
	regexp.MustCompile("Bracer 1"),
	regexp.MustCompile("Boots 3"),
	regexp.MustCompile("Boots 2"),
	regexp.MustCompile("Boots 1"),
	regexp.MustCompile("zOLD"),
	regexp.MustCompile("30 Epic"),
	regexp.MustCompile("Indalamar"),
	regexp.MustCompile("QR XXXX"),
	regexp.MustCompile("Deprecated: Keanna"),
	regexp.MustCompile("90 Epic"),
	regexp.MustCompile("66 Epic"),
	regexp.MustCompile("63 Blue"),
	regexp.MustCompile("90 Green"),
	regexp.MustCompile("63 Green"),
}

// allowList allows overriding to allow an item
var allowList = map[int]bool{
	9449:  true, // Manual Crowd Pummeler
	11815: true, // Hand of Justice
	12590: true, // Felstriker
	12632: true, // Storm Gauntlets
	15808: true, // Fine Light Crossbow (for hunter testing).
	17111: true, // Blazefury Medallion
	17112: true, // Empyrean Demolisher
	19808: true, // Rockhide Strongfish
	20966: true, // Jade Pendant of Blasting
	22395: true, // Totem of Rage
	23198: true, // Idol of Brutality
	23835: true, // Gnomish Poultryizer
	23836: true, // Goblin Rocket Launcher
	24114: true, // Braided Eternium Chain
	27947: true, // Totem of Impact
	28041: true, // Bladefist's Breadth
	28785: true, // The Lightning Capacitor
	31139: true, // Fist of Reckoning
	31149: true, // Gloves of Pandemonium
	31193: true, // Blade of Unquenched Thirst
	32508: true, // Necklace of the Deep
	33135: true, // Falling Star
	33140: true, // Blood of Amber
	33143: true, // Stone of Blades
	33144: true, // Facet of Eternity
	6360:  true, // Steelscale Crushfish
	8345:  true, // Wolfshead Helm
	28032: true, // Delicate Green Poncho
	32387: true, // Idol of the Raven Goddess
	32658: true, // Badge of Tenacity
	33504: true, // Libram of Divine Purpose
	33829: true, // Hex Shrunken Head

	15056: true, // Stormshroud Armor
	15057: true, // Stormshroud Pants
	15058: true, // Stormshroud Shoulders
	21278: true, // Stormshroud Gloves
}
