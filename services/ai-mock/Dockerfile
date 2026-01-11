# AI Mock Service - Simulacija LLM za MVP demo
# Za produkciju: Zameniti sa kontejnerom koji hostuje pravi model

FROM python:3.11-slim

LABEL maintainer="ARGUS <argus@lisenisvega.rs>"
LABEL description="AI Mock Service for anomaly detection demo"

WORKDIR /app

# Instaliraj zavisnosti
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Kopiraj kod
COPY main.py .

# Non-root user za bezbednost
RUN useradd -m -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:5000/health')" || exit 1

EXPOSE 5000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "5000"]
