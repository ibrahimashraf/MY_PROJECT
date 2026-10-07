import re
from anymd_plus import strip_edges
K = r"D:\bench_keith\chunk_00_0001-0110.pdf.anymd.md"
md = open(K, encoding="utf-8").read()
out = strip_edges(md)
print("headers:", md.count("Rigging Engineering Basics"), "->", out.count("Rigging Engineering Basics"))
print("ocr markers:", md.count("<!-- OCR text -->"), "->", out.count("<!-- OCR text -->"))
print("page markers:", len(re.findall(r"<!-- page \d+ -->", md)), "->",
      len(re.findall(r"<!-- page \d+ -->", out)))
# where does the header sit?
i = md.find("Rigging Engineering Basics")
print("ctx:", repr(md[max(0, i - 120):i + 60]))
