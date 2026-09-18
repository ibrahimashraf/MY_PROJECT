import pypdf
import re

def dump_pages(pdf_path, pages, out_name):
    reader = pypdf.PdfReader(pdf_path)
    text = ""
    for p in pages:
        if p <= len(reader.pages):
            text += f"\n=== PAGE {p} ===\n"
            text += reader.pages[p-1].extract_text() or ""
    with open(out_name, "w", encoding="utf-8") as f:
        f.write(text)
    print(f"Wrote {out_name}")

# BS 7121-1: Table 2 (p17), Wind (p18, p47-50), Multiple lifting (p44-46)
dump_pages(r"c:\MY_PROJECT\standards\BS 7121-1-2016.pdf", [17, 18, 44, 45, 46, 47, 48, 49, 50], "bs7121_1_details.txt")

# BS 7121-2-1: Thorough exam intervals, major review
dump_pages(r"c:\MY_PROJECT\standards\BS 7121-2-1-2012.pdf", [18, 19, 20, 21, 22, 23, 24, 25], "bs7121_2_1_details.txt")

# BS 7121-13: Gantry differential height, slopes, deflections
dump_pages(r"c:\MY_PROJECT\standards\BS 7121-13-2009.pdf", [21, 22, 23, 24, 30, 31, 32, 33], "bs7121_13_details.txt")

# BS 7121-14: Pipelayers tipping, slope
dump_pages(r"c:\MY_PROJECT\standards\BS 7121-14-2005.pdf", [8, 9, 10, 11, 12, 13], "bs7121_14_details.txt")

# BS 7121-5: Tower cranes wind, out of service, climbing, clearances
dump_pages(r"c:\MY_PROJECT\standards\BS 7121-5-2019.pdf", [11, 12, 13, 14, 25, 26, 27, 28, 29, 30, 45, 46], "bs7121_5_details.txt")
