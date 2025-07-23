from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import gensim
from gensim import corpora
from gensim.models import TfidfModel
import pickle

app = FastAPI()

# Load the pre-trained model and dictionary at startup
model_path = 'tfidf_model.pkl'
dict_path = 'dictionary.pkl'
try:
    with open(model_path, 'rb') as f:
        tfidf = pickle.load(f)
    with open(dict_path, 'rb') as f:
        dictionary = pickle.load(f)
except FileNotFoundError:
    # Fallback for model creation if not found
    documents = ["dummy text"]
    texts = [[word for word in document.lower().split()] for document in documents]
    dictionary = corpora.Dictionary(texts)
    corpus = [dictionary.doc2bow(text) for text in texts]
    tfidf = TfidfModel(corpus)

class EmbedRequest(BaseModel):
    text: str

@app.get("/health")
def health_check():
    return {"status": "ok"}

@app.post("/embed")
def create_embedding(request: EmbedRequest):
    try:
        # Preprocess the text
        processed_text = [word for word in request.text.lower().split()]
        bow = dictionary.doc2bow(processed_text)

        # Generate embedding
        embedding = tfidf[bow]

        # Convert to dense vector
        dense_embedding = gensim.matutils.sparse2full(embedding, 384).tolist()

        return {"embedding": dense_embedding}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
