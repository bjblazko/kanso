// SPDX-License-Identifier: Apache-2.0

package devseed

const welcomeBody = "Diese Daten sind erfunden und nur zum Ausprobieren da.\n\n" +
	"## Wer darf was\n\n" +
	"| Person | Rolle |\n|---|---|\n" +
	"| Anna | Admin |\n| Ben | Owner von *Haus*, Gruppe *familie* |\n" +
	"| Lena | Owner von *Lenas Ecke*, Gruppe *familie*, liest *Haus* |\n" +
	"| Oma Gerda | liest *Familie* |\n| Gast | sieht keinen Bereich |\n\n" +
	"## Ablauf am Samstag\n\n" +
	"```mermaid\nflowchart LR\n  A[Einkaufen] --> B[Kochen]\n  B --> C[Essen]\n  C --> D{Wetter gut?}\n" +
	"  D -->|ja| E[Spaziergang]\n  D -->|nein| F[Spieleabend]\n```\n"

const birthdaysBody = "- **Oma Gerda:** 14. März\n- **Lena:** 2. Juli\n- **Ben:** 23. September\n- **Anna:** 9. Dezember\n"

const heatingBody = "Gasbrennwerttherme im Keller.\n\n" +
	"| Was | Wann |\n|---|---|\n| Wartung | jährlich im September |\n| Wasser nachfüllen | unter 1,2 bar |\n\n" +
	"- [x] Wartung 2026 erledigt\n- [ ] Heizkörper entlüften\n"

const booksBody = "1. Ronja Räubertochter\n2. Die unendliche Geschichte\n3. Momo\n"
