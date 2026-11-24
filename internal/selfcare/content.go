// SPDX-License-Identifier: Apache-2.0

package selfcare

// Text is one piece of copy in German and English.
type Text struct {
	De string `json:"de"`
	En string `json:"en"`
}

// In returns the text in lang; German is the default.
func (t Text) In(lang string) string {
	if lang == "en" && t.En != "" {
		return t.En
	}
	return t.De
}

// Tip is a first-aid step: small, done in a minute or two, no tools.
type Tip struct {
	ID          string
	Title, Body Text
	// Gentle tips ask almost nothing; they are what an empty battery gets.
	Gentle bool
}

// Exercise is a guided practice reached on purpose: an intro and steps.
type Exercise struct {
	ID          string
	Title, Lead Text
	Steps       []Text
	// Kind is "steps", "breath" (paced counting) or "note" (ends in a private note).
	Kind string
}

var tips = []Tip{
	{"window", Text{"Ans Fenster", "To the window"}, Text{
		"Öffne ein Fenster, atme dreimal tief durch die Nase ein und schau eine Minute so weit nach draußen, wie es geht. Das Handy bleibt liegen.",
		"Open a window, breathe in deeply through your nose three times and look as far outside as you can for a minute. Leave the phone where it is."}, true},
	{"water", Text{"Ein Glas Wasser", "A glass of water"}, Text{
		"Hol dir ein Glas Wasser und trink es langsam, Schluck für Schluck.",
		"Get a glass of water and drink it slowly, sip by sip."}, true},
	{"shoulders", Text{"Schultern runter", "Shoulders down"}, Text{
		"Zieh die Schultern zu den Ohren, halte kurz und lass sie fallen. Dreimal. Dann den Kiefer locker lassen.",
		"Pull your shoulders up to your ears, hold for a moment and let them drop. Three times. Then let your jaw go loose."}, true},
	{"slow", Text{"Etwas langsam tun", "Do one thing slowly"}, Text{
		"Such dir eine Kleinigkeit und mach sie halb so schnell wie sonst: Hände waschen, Tee eingießen, eine Treppe gehen.",
		"Pick a small thing and do it at half your usual speed: washing your hands, pouring tea, taking the stairs."}, false},
	{"song", Text{"Ein Lied", "One song"}, Text{
		"Mach ein Lied an, das dich verlässlich aufrichtet, und tu drei Minuten lang nichts anderes. Mitsummen erlaubt.",
		"Put on a song that reliably lifts you and do nothing else for three minutes. Humming along is allowed."}, false},
	{"see", Text{"Fünf Dinge sehen", "See five things"}, Text{
		"Benenne leise fünf Dinge, die du gerade siehst, jeweils mit ihrer Farbe.",
		"Quietly name five things you can see right now, each with its colour."}, true},
	{"walk", Text{"Einmal um den Block", "Around the block"}, Text{
		"Zieh Schuhe an und geh fünf Minuten raus, ohne Ziel und ohne Kopfhörer.",
		"Put on your shoes and go out for five minutes, with no goal and no headphones."}, false},
	{"good", Text{"Eine gute Sache", "One good thing"}, Text{
		"Schreib einen Satz auf, was heute schon gut war. Ein Satz genügt.",
		"Write down one sentence about something that went well today. One sentence is enough."}, false},
	{"stretch", Text{"Strecken", "Stretch"}, Text{
		"Steh auf, streck die Arme so hoch es geht, dann lass den Oberkörper locker nach vorn hängen. Langsam wieder aufrichten.",
		"Stand up, reach your arms as high as you can, then let your upper body hang loosely forward. Roll up slowly."}, false},
	{"breath", Text{"Länger aus als ein", "Out longer than in"}, Text{
		"Atme bis vier ein und bis sechs aus. Sechsmal.",
		"Breathe in for four and out for six. Six times."}, true},
	{"hands", Text{"Warme Hände", "Warm hands"}, Text{
		"Reib die Handflächen kräftig aneinander, bis sie warm sind, und leg sie dir über die Augen oder in den Nacken.",
		"Rub your palms together hard until they are warm, then rest them over your eyes or on your neck."}, true},
	{"tidy", Text{"Eine Ecke Ordnung", "One tidy corner"}, Text{
		"Räum genau eine kleine Fläche auf, etwa den Platz direkt vor dir. Mehr nicht.",
		"Tidy exactly one small area, such as the space right in front of you. Nothing more."}, false},
	{"message", Text{"Jemandem schreiben", "Message someone"}, Text{
		"Schick einer Person, die dir guttut, eine kurze Nachricht. Ein Anlass ist nicht nötig.",
		"Send a short message to someone who is good for you. No reason needed."}, false},
	{"no", Text{"Ein Nein für heute", "One no for today"}, Text{
		"Such dir eine Sache aus, die heute nicht mehr passieren muss, und streich sie.",
		"Pick one thing that does not have to happen today any more, and cross it off."}, true},
	{"tenyears", Text{"In zehn Jahren?", "In ten years?"}, Text{
		"Frag dich: Werde ich mich in zehn Jahren daran erinnern? Wenn nicht: Wie viel Kraft soll es heute bekommen?",
		"Ask yourself: will I remember this in ten years? If not, how much energy should it get today?"}, false},
	{"friend", Text{"Wie zu einem Freund", "As to a friend"}, Text{
		"Was würdest du einem guten Freund in deiner Lage sagen? Sag es dir selbst, gern halblaut.",
		"What would you say to a good friend in your place? Say it to yourself, out loud if you like."}, false},
	{"feet", Text{"Füße spüren", "Feel your feet"}, Text{
		"Stell beide Füße fest auf den Boden, spür Fersen und Zehen und drück sie zehn Sekunden sanft nach unten.",
		"Put both feet firmly on the floor, feel your heels and toes and press them gently down for ten seconds."}, true},
	{"cold", Text{"Kaltes Wasser", "Cold water"}, Text{
		"Halt die Handgelenke eine halbe Minute unter kaltes Wasser.",
		"Hold your wrists under cold water for half a minute."}, true},
}

var exercises = []Exercise{
	{ID: "breathe", Kind: "breath", Title: Text{"Atmen 4–6", "Breathing 4–6"},
		Lead: Text{"Eine Minute: länger aus- als einatmen beruhigt. Kanso zählt mit.", "One minute: breathing out longer than in calms you down. Kanso counts along."},
		Steps: []Text{{"Setz dich bequem hin, beide Füße am Boden.", "Sit comfortably, both feet on the floor."},
			{"Atme durch die Nase ein und zähl bis vier.", "Breathe in through your nose and count to four."},
			{"Atme durch den Mund aus und zähl bis sechs.", "Breathe out through your mouth and count to six."},
			{"Sechs Runden, dann kurz nachspüren.", "Six rounds, then notice how you feel."}}},
	{ID: "senses", Kind: "steps", Title: Text{"Sinne 5-4-3-2-1", "Senses 5-4-3-2-1"},
		Lead: Text{"Wenn der Kopf zu voll ist: mit den Sinnen zurück in den Raum.", "When your head is too full: back into the room through your senses."},
		Steps: []Text{{"Fünf Dinge, die du siehst.", "Five things you see."}, {"Vier Dinge, die du spürst.", "Four things you feel."},
			{"Drei Geräusche, die du hörst.", "Three sounds you hear."}, {"Zwei Dinge, die du riechst.", "Two things you smell."},
			{"Eine Sache, die du schmeckst.", "One thing you taste."}}},
	{ID: "window", Kind: "steps", Title: Text{"Fenster-Moment", "Window moment"},
		Lead: Text{"Zwei Minuten frische Luft, bewusst und ohne Nebenbei.", "Two minutes of fresh air, on purpose and without doing anything else."},
		Steps: []Text{{"Steh auf und öffne ein Fenster.", "Stand up and open a window."}, {"Spür die Luft auf der Haut.", "Feel the air on your skin."},
			{"Lass die Schultern sinken, atme tief aus.", "Let your shoulders drop, breathe out deeply."},
			{"Schau auf den entferntesten Punkt, den du siehst.", "Look at the farthest point you can see."},
			{"Zähl langsam bis fünfzehn.", "Count slowly to fifteen."}}},
	{ID: "body", Kind: "steps", Title: Text{"Körper-Check", "Body check"},
		Lead: Text{"Von oben nach unten: wahrnehmen, nicht bewerten, ausatmen.", "From top to bottom: notice, do not judge, breathe out."},
		Steps: []Text{{"Stirn und Augen: Ist da Spannung? Lass sie weich werden.", "Forehead and eyes: any tension? Let them soften."},
			{"Kiefer: Zähne auseinander, Zunge locker.", "Jaw: teeth apart, tongue loose."},
			{"Schultern und Nacken: sinken lassen.", "Shoulders and neck: let them sink."},
			{"Bauch: Wie atmet er gerade? Nichts ändern, nur merken.", "Belly: how is it breathing? Change nothing, just notice."},
			{"Hände: öffnen, schließen, ruhen lassen.", "Hands: open, close, let them rest."}}},
	{ID: "kind", Kind: "steps", Title: Text{"Freundlich mit mir", "Kind to myself"},
		Lead: Text{"Für Momente, in denen die innere Stimme streng wird.", "For moments when the inner voice gets harsh."},
		Steps: []Text{{"Bemerke den Satz, den du dir gerade sagst.", "Notice the sentence you are telling yourself."},
			{"Würdest du ihn einem Menschen sagen, den du magst?", "Would you say it to someone you like?"},
			{"Sag es so, wie du es ihm sagen würdest.", "Say it the way you would say it to them."},
			{"Nenn eine Sache, die du heute geschafft hast, so klein sie ist.", "Name one thing you managed today, however small."}}},
	{ID: "checkin", Kind: "note", Title: Text{"Check-in", "Check-in"},
		Lead: Text{"Fünf Minuten für die Frage, wie es dir gerade geht, und einen kleinen nächsten Schritt.", "Five minutes for how you are right now, and one small next step."},
		Steps: []Text{{"Wie voll ist dein Akku, und woran merkst du das?", "How full is your battery, and how can you tell?"},
			{"Welches Gefühl ist gerade am lautesten? Benenne es ohne Urteil.", "Which feeling is loudest right now? Name it without judging."},
			{"Was kostet dich im Moment am meisten Kraft?", "What takes the most energy from you at the moment?"},
			{"Was würde dir jetzt guttun, klein oder groß?", "What would do you good now, small or big?"},
			{"Heute sage ich Nein zu … verschiebe … bitte um Hilfe bei … nehme mir zehn Minuten für …", "Today I say no to … put off … ask for help with … take ten minutes for …"}}},
	{ID: "good", Kind: "note", Title: Text{"Eine gute Sache", "One good thing"},
		Lead:  Text{"Eine Zeile über etwas, das heute gut war. Kein Tagebuch, eine Zeile.", "One line about something that was good today. Not a diary, one line."},
		Steps: []Text{{"Was war heute gut, gelungen oder schön?", "What was good, went well or was lovely today?"}}},
}
