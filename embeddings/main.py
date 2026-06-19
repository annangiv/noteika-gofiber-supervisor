from fastapi import FastAPI
from pydantic import BaseModel
from fastembed import TextEmbedding
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("embeddings")

app = FastAPI(title="Noteika Embeddings Service")

logger.info("Initializing FastEmbed Model (BAAI/bge-small-en-v1.5)...")
# Initialize model globally (loads from local cached file if pre-downloaded)
model = TextEmbedding()
logger.info("FastEmbed model loaded and ready!")

class EmbedRequest(BaseModel):
    text: str

@app.post("/embed")
def embed_text(payload: EmbedRequest):
    logger.info(f"Generating embedding for text: {payload.text[:30]}...")
    # model.embed returns a generator of numpy arrays
    embeddings = list(model.embed([payload.text]))
    embedding_list = embeddings[0].tolist()
    return {"embedding": embedding_list}

@app.get("/health")
def health_check():
    return {"status": "healthy"}
