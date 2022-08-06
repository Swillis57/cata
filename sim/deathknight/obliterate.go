package deathknight

import (
	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/proto"
	"github.com/wowsims/wotlk/sim/core/stats"
)

// TODO: Cleanup obliterate the same way we did for plague strike
var ObliterateActionID = core.ActionID{SpellID: 51425}
var ObliterateMHOutcome = core.OutcomeMiss
var ObliterateOHOutcome = core.OutcomeMiss

func (dk *Deathknight) newObliterateHitSpell(isMH bool) *RuneSpell {
	diseaseConsumptionChance := 1.0
	if dk.Talents.Annihilation == 1 {
		diseaseConsumptionChance = 0.67
	} else if dk.Talents.Annihilation == 2 {
		diseaseConsumptionChance = 0.34
	} else if dk.Talents.Annihilation == 3 {
		diseaseConsumptionChance = 0.0
	}

	bonusBaseDamage := dk.sigilOfAwarenessBonus(dk.Obliterate)
	weaponBaseDamage := core.BaseDamageFuncMeleeWeapon(core.MainHand, true, 584.0+bonusBaseDamage, 0.8, true)
	if !isMH {
		weaponBaseDamage = core.BaseDamageFuncMeleeWeapon(core.OffHand, true, 584.0+bonusBaseDamage, 0.8*dk.nervesOfColdSteelBonus(), true)
	}

	diseaseMulti := dk.diseaseMultiplier(0.125)

	effect := core.SpellEffect{
		BonusCritRating:  (dk.rimeCritBonus() + dk.subversionCritBonus() + dk.annihilationCritBonus() + dk.scourgeborneBattlegearCritBonus()) * core.CritRatingPerCritChance,
		DamageMultiplier: core.TernaryFloat64(dk.HasMajorGlyph(proto.DeathknightMajorGlyph_GlyphOfObliterate), 1.25, 1.0) * dk.scourgelordsBattlegearDamageBonus(dk.Obliterate),
		ThreatMultiplier: 1,

		BaseDamage: core.BaseDamageConfig{
			Calculator: func(sim *core.Simulation, hitEffect *core.SpellEffect, spell *core.Spell) float64 {
				return weaponBaseDamage(sim, hitEffect, spell) *
					(1.0 + dk.countActiveDiseases(hitEffect.Target)*diseaseMulti) *
					dk.RoRTSBonus(hitEffect.Target) *
					dk.mercilessCombatBonus(sim)
			},
			TargetSpellCoefficient: 1,
		},

		OnSpellHitDealt: func(sim *core.Simulation, spell *core.Spell, spellEffect *core.SpellEffect) {
			if isMH {
				ObliterateMHOutcome = spellEffect.Outcome
			} else {
				ObliterateOHOutcome = spellEffect.Outcome
			}

			if sim.RandomFloat("Annihilation") < diseaseConsumptionChance {
				dk.FrostFeverDisease[spellEffect.Target.Index].Deactivate(sim)
				dk.BloodPlagueDisease[spellEffect.Target.Index].Deactivate(sim)
			}

			if sim.RandomFloat("Rime") < dk.rimeHbChanceProc() {
				dk.RimeAura.Activate(sim)
			}
		},
	}

	dk.threatOfThassarianProcMasks(isMH, &effect, true, false, func(outcomeApplier core.OutcomeApplier) core.OutcomeApplier {
		return outcomeApplier
	})

	return dk.RegisterSpell(nil, core.SpellConfig{
		ActionID:     ObliterateActionID.WithTag(core.TernaryInt32(isMH, 1, 2)),
		SpellSchool:  core.SpellSchoolPhysical,
		Flags:        core.SpellFlagMeleeMetrics,
		ApplyEffects: core.ApplyEffectFuncDirectDamage(effect),
	})
}

func (dk *Deathknight) registerObliterateSpell() {
	dk.ObliterateMhHit = dk.newObliterateHitSpell(true)
	dk.ObliterateOhHit = dk.newObliterateHitSpell(false)

	amountOfRunicPower := 15.0 + 2.5*float64(dk.Talents.ChillOfTheGrave) + dk.scourgeborneBattlegearRunicPowerBonus()
	baseCost := float64(core.NewRuneCost(uint8(amountOfRunicPower), 0, 1, 1, 0))
	rs := &RuneSpell{}
	dk.Obliterate = dk.RegisterSpell(rs, core.SpellConfig{
		ActionID:     ObliterateActionID.WithTag(3),
		Flags:        core.SpellFlagNoMetrics | core.SpellFlagNoLogs,
		SpellSchool:  core.SpellSchoolPhysical,
		ResourceType: stats.RunicPower,
		BaseCost:     baseCost,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost,
				GCD:  core.GCDDefault,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				cast.GCD = dk.getModifiedGCD()
			},
		},
		ApplyEffects: dk.withRuneRefund(rs, core.SpellEffect{
			ProcMask:         core.ProcMaskEmpty,
			ThreatMultiplier: 1,

			OutcomeApplier: dk.OutcomeFuncAlwaysHit(),

			OnSpellHitDealt: func(sim *core.Simulation, spell *core.Spell, spellEffect *core.SpellEffect) {
				dk.threatOfThassarianProc(sim, spellEffect, dk.ObliterateMhHit, dk.ObliterateOhHit)
				dk.LastOutcome = spellEffect.Outcome
			},
		}, false),
	})
}

func (dk *Deathknight) CanObliterate(sim *core.Simulation) bool {
	return dk.CastCostPossible(sim, 0.0, 0, 1, 1) && dk.Obliterate.IsReady(sim)
}

func (dk *Deathknight) CastObliterate(sim *core.Simulation, target *core.Unit) bool {
	if dk.Obliterate.IsReady(sim) {
		return dk.Obliterate.Cast(sim, target)
	}
	return false
}
