from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import mistune

app = FastAPI()

# Create a mistune renderer with GitHub-flavored markdown plugins
renderer = mistune.create_markdown(
    renderer='html',
    escape=False,
    plugins=['strikethrough', 'footnotes', 'table', 'url', 'task_lists']
)

class MarkupRequest(BaseModel):
    text: str

@app.post("/markup")
def convert_to_html(request: MarkupRequest):
    try:
        html = renderer(request.text)
        return {"html": html}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
