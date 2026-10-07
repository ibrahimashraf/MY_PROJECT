"""anymd_plus GUI: pick a file or folder, process, done. Stdlib only (tkinter).
PDFs -> convert (anymd, chunked if >512MB) -> clean [+ reshape if Arabic].
.md  -> clean [+ reshape if Arabic]. Threaded; log pane; opens output folder."""
import os
import queue
import shutil
import subprocess
import threading
import tkinter as tk
from tkinter import filedialog, messagebox, ttk

from anymd_plus import arabic_ratio, arabic_reshape, clean, stats

NPX = shutil.which("npx.cmd") or "npx"


def convert_pdf(pdf, out_md, ocr, log):
    from anymd_plus import chunk_convert
    log("convert: " + os.path.basename(pdf))
    md = chunk_convert(pdf, ocr=ocr)
    with open(out_md, "w", encoding="utf-8") as f:
        f.write(md)
    return md


def process_file(path, outdir, ocr, log):
    base = os.path.splitext(os.path.basename(path))[0]
    if path.lower().endswith(".pdf"):
        raw_md = os.path.join(outdir, base + ".raw.md")
        md = convert_pdf(path, raw_md, ocr, log)
    else:
        md = open(path, encoding="utf-8").read()
    md = clean(md)
    if arabic_ratio(md) >= 0.15:
        log("arabic detected -> reshape")
        md = arabic_reshape(md)
    final = os.path.join(outdir, base + ".clean.md")
    with open(final, "w", encoding="utf-8") as f:
        f.write(md)
    s = stats(md)
    log("%s -> words=%d tok~%d tables=%d heads=%d garbage=%.3f"
        % (os.path.basename(final), s["words"], s["tokens_est"],
           s["tables"], s["headings"], s["garbage_ratio"]))
    return final


class App(tk.Tk):
    def __init__(self):
        super().__init__()
        self.title("anymd_plus")
        self.geometry("640x460")
        self.paths, self.outdir = [], tk.StringVar(value=os.path.expanduser("~\\Documents\\anymd_out"))
        self.ocr = tk.BooleanVar(value=False)
        self.q = queue.Queue()
        bar = ttk.Frame(self)
        bar.pack(fill="x", padx=8, pady=6)
        ttk.Button(bar, text="Add files", command=self.add_files).pack(side="left")
        ttk.Button(bar, text="Add folder", command=self.add_folder).pack(side="left", padx=4)
        ttk.Button(bar, text="Clear", command=lambda: (self.paths.clear(), self.refresh())).pack(side="left")
        mid = ttk.Frame(self)
        mid.pack(fill="x", padx=8)
        ttk.Label(mid, text="Output:").pack(side="left")
        ttk.Entry(mid, textvariable=self.outdir, width=48).pack(side="left", padx=4)
        ttk.Button(mid, text="…", width=3, command=self.pick_out).pack(side="left")
        ttk.Checkbutton(mid, text="OCR scans", variable=self.ocr).pack(side="left", padx=8)
        self.lst = tk.Listbox(self, height=6)
        self.lst.pack(fill="x", padx=8, pady=6)
        go = ttk.Frame(self)
        go.pack(fill="x", padx=8)
        self.run_btn = ttk.Button(go, text="Process", command=self.go)
        self.run_btn.pack(side="left")
        ttk.Button(go, text="Open output", command=self.open_out).pack(side="left", padx=4)
        self.prog = ttk.Progressbar(go, mode="indeterminate")
        self.logw = tk.Text(self, height=12, state="disabled")
        self.logw.pack(fill="both", expand=True, padx=8, pady=6)
        self.after(150, self.pump)

    def log(self, msg):
        self.q.put(msg)

    def pump(self):
        while not self.q.empty():
            self.logw.configure(state="normal")
            self.logw.insert("end", self.q.get() + "\n")
            self.logw.configure(state="disabled")
            self.logw.see("end")
        self.after(150, self.pump)

    def refresh(self):
        self.lst.delete(0, "end")
        for p in self.paths:
            self.lst.insert("end", p)

    def add_files(self):
        for f in filedialog.askopenfilenames(
                filetypes=[("PDF/Markdown", "*.pdf *.md"), ("All", "*.*")]):
            if f not in self.paths:
                self.paths.append(f)
        self.refresh()

    def add_folder(self):
        d = filedialog.askdirectory()
        if not d:
            return
        for root, _, fs in os.walk(d):
            for f in fs:
                if f.lower().endswith((".pdf", ".md")):
                    p = os.path.join(root, f)
                    if p not in self.paths:
                        self.paths.append(p)
        self.refresh()

    def pick_out(self):
        d = filedialog.askdirectory(initialdir=self.outdir.get())
        if d:
            self.outdir.set(d)

    def open_out(self):
        os.startfile(self.outdir.get())

    def go(self):
        if not self.paths:
            messagebox.showinfo("anymd_plus", "Add files or a folder first.")
            return
        os.makedirs(self.outdir.get(), exist_ok=True)
        self.run_btn.configure(state="disabled")
        self.prog.start()
        threading.Thread(target=self.work, daemon=True).start()

    def work(self):
        ocr, outdir = self.ocr.get(), self.outdir.get()
        ok, fail = 0, 0
        for p in list(self.paths):
            try:
                process_file(p, outdir, ocr, self.log)
                ok += 1
            except Exception as e:
                self.log("FAIL %s: %s" % (os.path.basename(p), str(e)[:200]))
                fail += 1
        self.log("done: %d ok, %d failed -> %s" % (ok, fail, outdir))
        self.prog.stop()
        self.run_btn.configure(state="normal")


if __name__ == "__main__":
    App().mainloop()
