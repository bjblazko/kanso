# Hauptmenü anpassbar, Pomodoro „Fokus“

**Status:** freigegeben (Owner, 2026-11-24)

## Warum

Es kommen weitere Hauptfunktionen. Das Menü (die Leiste, ADR-005) muss mit
ihnen wachsen, und jede Person soll pro Gerät Funktionen ausblenden (z. B.
Wiki) und umsortieren können. Als erste neue Funktion dient ein
Pomodoro-Timer zum Testen.

## Entscheidungen des Owners

- Einstellungen pro Gerät und Person, im Browser gespeichert, **ohne Cookie**.
- Benachrichtigungen gibt es **nur bei offenem Tab**, auch in Zukunft. Web
  Push entfällt (ADR-021).
- Desktop: Die Leiste scrollt senkrecht. Handy: „Mehr“ (…).

## Menü

- Die Einträge sind serverseitig eine Liste (`railItems`: key, href, icon,
  label), in dieser Standardreihenfolge: Start, Notizen, Todos, Aufgaben,
  Wiki, Küche, Fokus.
- `localStorage["kanso.menu.<sub>"] = {"order":[keys],"hidden":[keys]}`.
  Unbekannte Schlüssel fallen weg. Neue Funktionen hängen sich in der
  Standardreihenfolge hinten an. „Start“ lässt sich nicht ausblenden.
- `menu.js` ist ein klassisches, synchrones Skript direkt nach der Leiste.
  Es ordnet um und blendet aus, bevor gezeichnet wird. Die CSP bleibt `self`.
- Handy: Bei mehr als 5 sichtbaren Einträgen stehen die ersten 4 in der Leiste
  und dazu „Mehr“. „Mehr“ klappt eine Liste mit dem Rest und
  „Menü anpassen“ auf. Ist die aktuelle Seite darin, ist „Mehr“ markiert.
- Desktop: alle sichtbaren Einträge, die Leiste scrollt senkrecht, kein „Mehr“.
- Seite `/me/menu` (Benutzermenü und „Mehr“): pro Eintrag ein Schalter
  „Im Menü“ und ↑/↓. Dazu „Zurücksetzen“. Änderungen gelten sofort, auch für
  die Leiste auf derselben Seite.
- Ohne JavaScript zeigt das Menü die Standardreihenfolge.
- Ausgeblendete Funktionen bleiben per Link erreichbar.

## Fokus (Pomodoro)

- `/focus`, Modul `focus`, Icon `timer`.
- Phasen: Fokus 25, Pause 5, nach 4 Fokusrunden lange Pause 15 Minuten. Die
  Dauern sind pro Gerät einstellbar (1–120 min) und liegen in
  `localStorage["kanso.focus.<sub>"]`, zusammen mit dem laufenden Zustand
  (Phase, Runde, Endzeitpunkt oder Rest bei Pause).
- Bedienung: Start/Pause (eine Taste, Primäraktion), Zurücksetzen,
  Überspringen. Die nächste Phase wartet auf Start.
- Die Zeit zählt gegen den Endzeitpunkt. Neu laden setzt fort, gedrosselte
  Hintergrund-Tabs bleiben genau. Der Tab-Titel zeigt „12:34 · Fokus“.
- Am Ende: ein Ton per Web Audio (keine Datei) und, wenn erlaubt, eine
  `Notification`. Die Erlaubnis wird beim ersten Start angefragt, eine
  Ablehnung lässt nur den Ton.
- Kein Server-Zustand, nichts in Git.

## Ansätze

- **localStorage + synchrones Skript (gewählt):** pro Gerät, ohne Cookie.
- Cookie und Server-Rendering: kein Springen, aber vom Owner nicht gewollt.
- Server-Profil: gleich auf allen Geräten, aber nicht pro Gerät.

## Nicht-Ziele

Web Push, Statistik der Pomodoros, Verknüpfung mit Aufgaben, Synchronisation
zwischen Geräten.
