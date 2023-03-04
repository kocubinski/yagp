package main

import (
	"bufio"
	"github.com/gdamore/tcell/v2"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type spellActor func(screenOpts, spell)

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

var spellTriggers = []spellTrigger{
	newTrigger(`^You begin casting (.*)$`, func(s screenOpts, spell spell) {
		s.drawLine(tcell.StyleDefault, spell.name)
	}),
}

type spell struct {
	id            int64
	name          string
	effectYou     string
	effectOther   string
	effectWornOff string
	castTime      int64
	duration      int64
}

type spellBook struct {
	byName        map[string]spell
	bySelfEffect  map[string]spell
	byOtherEffect map[string]spell
	byWornOff     map[string]spell
}

func (sb spellBook) matchTrigger(triggers []spellTrigger, line string) (spell, bool) {
	for _, t := range triggers {
		m := t.re.FindStringSubmatch(line)
		if len(m) > 0 {
			// since we're receiving against spellBook, assume this is a spell spellTrigger
			spellName := m[0]
			spell, ok := sb.byName[spellName]
			if !ok {
				log.Fatalf("spell not found: %s", spellName)
			}

			return spell, true
		}
	}

	return spell{}, false
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
		duration, _ := strconv.ParseInt(values[17], 10, 32)
		s := spell{
			id:            id,
			name:          strings.ToLower(values[1]),
			effectYou:     values[6],
			effectOther:   values[7],
			effectWornOff: values[8],
			castTime:      castTime,
			duration:      duration,
		}
		book.byName[s.name] = s
		book.bySelfEffect[s.effectYou] = s
		book.byOtherEffect[s.effectOther] = s
		book.byWornOff[s.effectWornOff] = s
	}

	return book
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
