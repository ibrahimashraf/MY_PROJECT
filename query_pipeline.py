import sys
import os
from fastembed import TextEmbedding
from qdrant_client import QdrantClient

def query(search_text: str):
    qdrant_path = os.path.join("pipeline_output", "qdrant_db")
    if not os.path.exists(qdrant_path):
        print("Error: Index not found. Run run_pipeline.py first.")
        sys.exit(1)

    client = QdrantClient(path=qdrant_path)
    embed_model = TextEmbedding(model_name="BAAI/bge-small-en-v1.5")
    query_vector = list(embed_model.embed([search_text]))[0]

    results = client.query_points(
        collection_name="pdf_docs",
        query=query_vector.tolist(),
        limit=3
    ).points

    print(f"\n=== Top Results for: '{search_text}' ===")
    for idx, hit in enumerate(results, 1):
        breadcrumb = hit.payload.get("breadcrumb", "Root")
        print(f"\n[{idx}] Score: {hit.score:.4f} | Clause: {breadcrumb}")
        print(hit.payload.get("text", "")[:400] + ("..." if len(hit.payload.get("text", "")) > 400 else ""))

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python query_pipeline.py \"your query\"")
        sys.exit(1)
    query(sys.argv[1])
