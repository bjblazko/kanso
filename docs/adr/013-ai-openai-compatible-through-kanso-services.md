# 013 — AI Through an OpenAI-Compatible API, Content Only Through Kanso's Services

**Status:** Accepted
**Created:** 2026-11-07
**Updated:** 2026-11-07

## Context

Kanso should use a language model, first to turn a recipe web page into a
recipe, later to tidy, summarise and search the wiki. The owner runs Ollama
on a machine that sleeps and is woken by a proxy, without authentication.
Kanso is a public project: others should be able to use a hosted service
such as OpenRouter with a token. The model must never get more than the
person asking may see, and must not change content on its own.

## Decision

- **Protocol:** the OpenAI-compatible chat completions API
  (`POST {base}/chat/completions`, optional bearer token, no streaming),
  called with the Go standard library. One connection, registered by an
  admin under "KI-Anbindung": base URL, model, optional token, on/off, and a
  connection test.
- **Secrets:** the configuration lives in `<state dir>/ai.json` (mode 0600),
  not in the data repository; the token is never shown again or logged.
  Kanso logs no text it sends or receives.
- **Generous time limit** (180 s): waking the machine, loading and answering.
- **Content access:** the model gets content only through Kanso's Go
  services, called with the rights of the person who triggered the
  function — the same checks as the web pages; never the file system. What
  the model produces becomes a draft the person confirms; it writes nothing
  by itself. Later features expose read-only services as tools.
- **Say where text goes:** every AI function names the host before sending.
- When an AI connection is on, every signed-in person may use it.

## Alternatives Considered

- **Ollama's own API:** simplest for the owner, but locks out OpenRouter,
  OpenAI, LM Studio and others; most of them speak the OpenAI API, Ollama too.
- **A provider SDK:** a new dependency for a single HTTP call.
- **Letting the model read files or a database directly:** no per-person
  rights, no read-only guarantee.
- **Configuration through environment variables:** fine for one machine,
  but the owner wanted admins to register the model in Kanso.

## Consequences

- Hosted providers process what is sent; the admin page says that a
  processing agreement with the provider may be needed.
- Long waits are normal with a sleeping local machine; pages say so.
- Recipe import first reads a page's own recipe data (JSON-LD) and asks the
  model only when there is none — so it also works without any AI.
