FROM python:3.9-slim

WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir fastapi uvicorn fastembed

# Pre-download and cache the default embedding model in the image
RUN python -c "from fastembed import TextEmbedding; TextEmbedding()"

# Copy application script
COPY main.py .

EXPOSE 8000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
