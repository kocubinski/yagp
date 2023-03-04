package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var CharacterLevel = float64(24)

type spellActor func(*appState, []string) bool

type spellTrigger struct {
	re    *regexp.Regexp
	actor spellActor
}

func newTrigger(re string, actor spellActor) spellTrigger {
	return spellTrigger{
		re:    regexp.MustCompile(re),
		actor: actor,
	}
}

func (sb spellBook) makeSpellTriggers() []spellTrigger {
	triggers := []spellTrigger{
		newTrigger(`You begin casting (.*)\.$`, func(s *appState, m []string) bool {
			spellName := m[1]
			if sp, ok := sb.byName[spellName]; ok {
				s.casting = &sp
				return true
			}
			return false
		}),
	}
	for k, s := range sb.byOtherEffect {
		if k == "" {
			continue
		}
		t := newTrigger(fmt.Sprintf(`(.*)(%s)$`, s.effectOther),
			func(s *appState, m []string) bool {
				target := m[1]
				sp, ok := s.spellBook.byOtherEffect[m[2]]
				if !ok {
					return false
				}

				if s.casting != nil && s.casting.id == sp.id {
					s.casting = nil
				} else {
					return false
				}

				s.timers = append(s.timers, timer{
					startedAt: time.Now(),
					duration:  spellDuration(sp, CharacterLevel),
					text:      fmt.Sprintf("%s - %s", sp.name, target),
				})

				return true
			})
		triggers = append(triggers, t)
	}

	return triggers
}

type spell struct {
	id            int64
	name          string
	effectYou     string
	effectOther   string
	effectWornOff string
	castTime      int64
	duration      int64
	formula       int64
}

type spellBook struct {
	byName        map[string]spell
	bySelfEffect  map[string]spell
	byOtherEffect map[string]spell
	byWornOff     map[string]spell
}

func newSpellBook() spellBook {
	f, err := os.Open("spells_us.txt")
	if err != nil {
		panic(err)
	}

	book := spellBook{
		byName:        make(map[string]spell),
		bySelfEffect:  make(map[string]spell),
		byOtherEffect: make(map[string]spell),
		byWornOff:     make(map[string]spell),
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		values := strings.Split(line, "^")
		id, _ := strconv.ParseInt(values[0], 10, 32)
		castTime, _ := strconv.ParseInt(values[13], 10, 32)
		formula, _ := strconv.ParseInt(values[16], 10, 32)
		duration, _ := strconv.ParseInt(values[17], 10, 32)
		s := spell{
			id:            id,
			name:          values[1],
			effectYou:     values[6],
			effectOther:   values[7],
			effectWornOff: values[8],
			castTime:      castTime,
			duration:      duration,
			formula:       formula,
		}
		book.byName[s.name] = s
		book.bySelfEffect[s.effectYou] = s
		book.byOtherEffect[s.effectOther] = s
		book.byWornOff[s.effectWornOff] = s
	}

	return book
}

func spellDuration(spell spell, level float64) int64 {
	ticks := float64(0)
	duration := float64(spell.duration)

	switch spell.formula {
	case 0:
		ticks = 0
	case 1:
		ticks = math.Ceil(level / 2)
		math.Min(ticks, duration)
	case 2:
		ticks = math.Ceil(level / 5 * 3)
		math.Min(ticks, duration)
	case 3:
		ticks = level * 30
		math.Min(ticks, duration)
	case 4:
		if duration == 0 {
			ticks = 50
		} else {
			ticks = duration
		}
	case 5:
		ticks = duration
		if ticks == 0 {
			ticks = 3
		}
	case 6:
		ticks = math.Ceil(level / 2)
		ticks = math.Min(ticks, duration)
	case 7:
		ticks = level
		ticks = math.Min(ticks, duration)
	case 8:
		ticks = level + 10
		ticks = math.Min(ticks, duration)
	case 9:
		ticks = level*2 + 10
		ticks = math.Min(ticks, duration)
	case 10:
		ticks = level*3 + 10
		ticks = math.Min(ticks, duration)
	case 11, 12, 15:
		ticks = duration
	case 50:
		ticks = 72000
	case 3600:
		if duration == 0 {
			ticks = 3600
		} else {
			ticks = duration
		}
	}

	return spell.duration
}

//def create_spell_book():
//""" Returns a dictionary of Spell by k, v -> spell_name, Spell() """
//spell_book = {}
//text_lookup_self = {}
//text_lookup_other = {}
//with open('data/spells/spells_us.txt') as spell_file:
//for line in spell_file:
//values = line.strip().split('^')
//spell = Spell(
//id=int(values[0]),
//name=values[1].lower(),
//effect_text_you=values[6],
//effect_text_other=values[7],
//effect_text_worn_off=values[8],
//aoe_range=int(values[10]),
//max_targets=(6 if int(values[10]) > 0 else 1),
//cast_time=int(values[13]),
//resist_type=int(values[85]),
//duration_formula=int(values[16]),
//pvp_duration_formula=int(values[181]),
//duration=int(values[17]),
//pvp_duration=int(values[182]),
//type=int(values[83]),
//spell_icon=int(values[144])
//)
//spell_book[values[1]] = spell
//text_lookup_self[spell.effect_text_you] = spell
//text_lookup_other[spell.effect_text_other] = spell
//return spell_book, text_lookup_self, text_lookup_other

/*
def get_spell_duration(spell, level):
    if spell.name in config.data['spells']['use_secondary']:
        formula, duration = spell.pvp_duration_formula, spell.pvp_duration
    elif config.data['spells']['use_secondary_all'] and spell.type == 0:
        formula, duration = spell.pvp_duration_formula, spell.pvp_duration
    else:
        formula, duration = spell.duration_formula, spell.duration

    spell_ticks = 0
    if formula == 0:
        spell_ticks = 0
    if formula == 1:
        spell_ticks = int(math.ceil(level / float(2.0)))
        spell_ticks = min(spell_ticks, duration)
    if formula == 2:
        spell_ticks = int(math.ceil(level / float(5.0) * 3))
        spell_ticks = min(spell_ticks, duration)
    if formula == 3:
        spell_ticks = int(level * 30)
        spell_ticks = min(spell_ticks, duration)
    if formula == 4:
        if duration == 0:
            spell_ticks = 50
        else:
            spell_ticks = duration
    if formula == 5:
        spell_ticks = duration
        if spell_ticks == 0:
            spell_ticks = 3
    if formula == 6:
        spell_ticks = int(math.ceil(level / float(2.0)))
        spell_ticks = min(spell_ticks, duration)
    if formula == 7:
        spell_ticks = level
        spell_ticks = min(spell_ticks, duration)
    if formula == 8:
        spell_ticks = level + 10
        spell_ticks = min(spell_ticks, duration)
    if formula == 9:
        spell_ticks = int((level * 2) + 10)
        spell_ticks = min(spell_ticks, duration)
    if formula == 10:
        spell_ticks = int(level * 3 + 10)
        spell_ticks = min(spell_ticks, duration)
    if formula == 11:
        spell_ticks = duration
    if formula == 12:
        spell_ticks = duration
    if formula == 15:
        spell_ticks = duration
    if formula == 50:
        spell_ticks = 72000
    if formula == 3600:
        if duration == 0:
            spell_ticks = 3600
        else:
            spell_ticks = duration
    return spell_ticks
*/
