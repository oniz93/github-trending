from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer

app = FastAPI()

# Load the pre-trained model at startup
model = SentenceTransformer('all-MiniLM-L6-v2')

class EmbedRequest(BaseModel):
    text: str

@app.get("/health")
def health_check():
    return {"status": "ok"}

@app.post("/embed")
def create_embedding(request: EmbedRequest):
    try:
        # Generate embedding
        embedding = model.encode(request.text, convert_to_tensor=False).tolist()

        return {"embedding": embedding}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))