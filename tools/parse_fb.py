import re
import html
import json

html_path = r'C:\Users\hima3\.gemini\antigravity-ide\brain\15751b1f-4c6b-4260-bdf7-b4f9719b2314\.system_generated\steps\1138\content.md'
with open(html_path, 'r', encoding='utf-8', errors='ignore') as f:
    text = f.read()

scripts = re.findall(r'<script[^>]*type="application/json"[^>]*>(.*?)</script>', text, re.DOTALL)

for s in scripts:
    if "تخيل" in s:
        data = json.loads(s)
        def find_all_strings(obj):
            if isinstance(obj, dict):
                for k, v in obj.items():
                    if isinstance(v, str) and ("تخيل" in v or "takeover" in v or "vulnerability" in v or "severity" in v):
                        print(f"Key: {k} -> Length: {len(v)}")
                        print(v[:500])
                        print("="*40)
                    else:
                        find_all_strings(v)
            elif isinstance(obj, list):
                for item in obj:
                    find_all_strings(item)
        find_all_strings(data)
