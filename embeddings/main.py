import re
import logging
from fastapi import FastAPI
from pydantic import BaseModel, Field
from fastembed import TextEmbedding
import yake

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("embeddings")

app = FastAPI(title="Noteika Embeddings Service")

logger.info("Initializing FastEmbed Model (BAAI/bge-small-en-v1.5)...")
model = TextEmbedding()
logger.info("FastEmbed model loaded and ready!")

kw_extractor = yake.KeywordExtractor(
    lan="en",
    n=2,
    dedupLim=0.85,
    top=6,
    features=None,
)

HASHTAG_RE = re.compile(r"#([a-zA-Z][a-zA-Z0-9_-]{1,31})")


def normalize_tag(raw: str) -> str:
    tag = raw.strip().lower().lstrip("#")
    tag = re.sub(r"[^a-z0-9_-]+", "-", tag)
    return tag.strip("-_")


class EmbedRequest(BaseModel):
    text: str


class TagSuggestRequest(BaseModel):
    title: str = ""
    body: str = ""
    existing_tags: list[str] = Field(default_factory=list)


@app.post("/embed")
def embed_text(payload: EmbedRequest):
    logger.info("Generating embedding for text: %s...", payload.text[:30])
    embeddings = list(model.embed([payload.text]))
    return {"embedding": embeddings[0].tolist()}


@app.post("/suggest-tags")
def suggest_tags(payload: TagSuggestRequest):
    text = f"{payload.title}\n{payload.body}".strip()
    logger.info("Suggesting tags for: %s...", text[:40])

    tags: set[str] = set()

    for tag in payload.existing_tags:
        normalized = normalize_tag(tag)
        if len(normalized) >= 2:
            tags.add(normalized)

    for match in HASHTAG_RE.findall(text):
        normalized = normalize_tag(match)
        if len(normalized) >= 2:
            tags.add(normalized)

    if len(text) >= 20:
        try:
            keywords = kw_extractor.extract_keywords(text)
            for kw, _score in keywords:
                normalized = normalize_tag(kw.replace(" ", "-"))
                if len(normalized) >= 2 and len(normalized) <= 32:
                    tags.add(normalized)
        except Exception as exc:
            logger.warning("YAKE tag extraction failed: %s", exc)

    # Drop overly generic tokens
    stop = {"note", "notes", "the", "and", "for", "with", "this", "that", "from"}
    tags = {t for t in tags if t not in stop}

    result = sorted(tags)[:8]
    return {"tags": result}


@app.get("/health")
def health_check():
    return {"status": "healthy"}
