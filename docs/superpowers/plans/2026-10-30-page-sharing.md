# Seiten einzeln freigeben Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Share one wiki page (and its subpages) with people or groups, with a role, as an exception to its space; recipients find it under "Für dich freigegeben".

**Architecture:** Page members live in the page frontmatter (`members`, same shape as `access.Member`) and in the wiki index. The web layer computes a page role = highest of the space role and the members of the page and its ancestors; every page access check, the search filter, the navigation tree and the move/promote targets use it. A share page `/p/<id>/share` edits the members.

**Tech Stack:** Go stdlib, `html/template`, Playwright.

**Spec:** `docs/superpowers/specs/2026-10-30-page-sharing-design.md`

## Global Constraints

- No new dependency; SPDX header in every new file; functions ~30 lines, gocyclo ≤ 10 (`~/go/bin/gocyclo -over 10`).
- Page shares only add rights; admins are owners everywhere (`access.Resolve`).
- Roles: viewer "Lesen", editor "Bearbeiten", owner "Verwalten" (on pages); same `access.Role` values.
- Who may change a page's shares: `access.Can(pageRole, access.ManageSpace)` (space owners, admins, page "Verwalten").
- Commit message for share changes: `Freigabe von „<Titel>“ geändert`, author = the person.
- UI copy de/en in `internal/i18n`; selects without button classes; Rams rules.
- Docs: CHANGELOG, ADR-009 "Page shares add rights", arc42 (ch. 5/6/9), feature doc `docs/features/2026-10-30 page-sharing.md`, `docs/design.md`.

## Review Focus

1. A recipient of one page asks for a sibling page, the space page, a sibling's attachment or history, or searches for a sibling's words → 404 / no hit.
2. A shared page is moved elsewhere in the space → its own shares move with it; a subpage moved out of the shared subtree is no longer visible to the recipient.
3. A recipient with Bearbeiten opens "Verschieben" or note → wiki page → only targets inside pages they may write; posting a target outside → 403.
4. The space owner removes the share → the page disappears from the recipient's "Für dich freigegeben" and opens as 404.
5. Saving a page (title/body) keeps its `members`.

---

### Task 1: wiki – members per page

**Files:** Modify `internal/wiki/tree.go` (PageMeta.Members), create `internal/wiki/shares.go`; test `internal/wiki/shares_test.go`.

**Interfaces (produces):**
- `PageMeta.Members []access.Member` (`yaml:"members,omitempty"`)
- `func (w *Wiki) Chain(id string) []PageMeta` — the page and its ancestors, page first
- `func (w *Wiki) SetMembers(a store.Author, id string, members []access.Member) error` — writes frontmatter, commits `Freigabe von „T“ geändert`, updates index
- `func (w *Wiki) Descendants(id string) int`
- `func (w *Wiki) Shared() []Page` — every page (Meta, Space, Dir, no Body) with non-empty Members

- [ ] Tests: `TestSetMembersCommitsAndIndexes`, `TestSaveKeepsMembers`, `TestChainIsPageThenAncestors`, `TestDescendants`, `TestMembersMoveWithPage` (Move keeps the page's own members; a moved child's Chain no longer contains the old parent), `TestShared`. → FAIL → implement → PASS → commit `feat(wiki): pages carry their own members`.

### Task 2: web – page role everywhere

**Files:** create `internal/web/page_access.go` (+ test `page_access_test.go`); modify `handlers_pages.go` (pageFor, createPage/newPageForm under a shared parent, moveTargets), `handlers_search.go` (filter by page), `handlers_notes_promote.go` (targets + check), `render.go` (`inSpace` prunes the tree), `handlers_spaces.go` unchanged.

**Interfaces (produces):**
- `func (s *Server) pageRole(p access.Principal, pg wiki.Page) access.Role` — max(space role, `access.Resolve(p, members of Chain, groups)`)
- `func (s *Server) visibleTree(p access.Principal, space string) []*wiki.Node` — full tree with space Read, else only subtrees whose root the person reads via shares

- [ ] Tests (Christina = viewer on page "Wunschliste" with child "Bücher", sibling "Geheim"): `TestShareGivesPageAndChildren`, `TestShareHidesSiblingsAndSpace` (page, attachment, history of sibling → 404; `/s/space` → 404), `TestShareReadOnly` (edit → 403), `TestShareEditorWritesInSubtree` (save 303, create child under shared page, move targets only inside, move outside → 403), `TestSearchOnlyShared`, `TestPromoteTargetsIncludeSharedWritable`. → FAIL → implement → PASS → commit `feat(wiki): page shares grant access to a page and its subpages`.

### Task 3: share page, audience line, link

**Files:** create `internal/web/handlers_page_share.go` (+ test), `web/templates/page_share.html`; modify `audience.go` (page audience adds page-share people; link to `/p/<id>/share` for who may manage), `handlers_pages.go` (pageView), server.go routes `GET|POST /p/{id}/share`, i18n.

**Interfaces:** form `action=add` (`kind` user|group, `who`, `role`) / `action=remove` (`index`); page shows inherited shares (from ancestors, read-only with link) and the hint "Gilt auch für N Unterseiten."

- [ ] Tests: `TestPageShareOnlyManagers` (editor 403, space owner ok, page "Verwalten" ok), `TestPageShareAddRemove` (commit, members), `TestPageShareHintAndInherited`, `TestPageAudienceNamesSharedPeople`, `TestPageAudienceLinksToPageShare`. → FAIL → implement → PASS → commit `feat(wiki): share a page from its page`.

### Task 4: "Für dich freigegeben"

**Files:** modify `render.go` (view.Shared), `layout.html` (sidenav section), `home.html`, `handlers_spaces.go` (home), page breadcrumbs for recipients (`page_view.html` uses Ancestors filtered to readable), CSS if needed, i18n.

**Interfaces:** `type sharedEntry struct { Node *wiki.Node; SpaceName, From string }`; `view.Shared []sharedEntry` — roots shared with the person in spaces they cannot read, with their subtrees.

- [ ] Tests: `TestSharedSectionInNav`, `TestSharedOnHome`, `TestSharedSectionHiddenWhenEmpty`, `TestBreadcrumbsStartAtSharedRoot`, `TestRemovedShareDisappears`. → FAIL → implement → PASS → Playwright `page-sharing.spec.js` (owner shares with hint, recipient finds it, sees child, not sibling) → commit `feat(wiki): pages shared with you have their own place`.

### Task 5: docs and verification

- [ ] ADR-009, arc42, feature doc, CHANGELOG, design.md; `just test`, `just lint`, gocyclo, full `just e2e`-equivalent run; commit `docs: page sharing`.
