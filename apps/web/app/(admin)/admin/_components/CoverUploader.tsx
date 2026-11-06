"use client";

import { useRef, useState } from "react";
import { API_URL, Icon, errMsg, useToast } from "./ui";

type Props = {
  coverUrl: string | null;
  onUploaded: (coverUrl: string, coverStorageKey: string) => void;
  onRemoved: () => void;
};

const MAX_SIZE = 5 * 1024 * 1024; // 5MB
const ACCEPTED = ["image/jpeg", "image/png", "image/webp"];
const ASPECT_WARN = 2 / 3; // ideal 2:3 (largura:altura)

export function CoverUploader({ coverUrl, onUploaded, onRemoved }: Props) {
  const toast = useToast();
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [dragOver, setDragOver] = useState(false);

  const validateFile = (file: File): string | null => {
    if (!ACCEPTED.includes(file.type)) return "Formato não suportado. Use JPG, PNG ou WebP.";
    if (file.size > MAX_SIZE) return "Arquivo maior que 5MB.";
    return null;
  };

  const handleFile = async (file: File) => {
    const err = validateFile(file);
    if (err) {
      toast.show(err, false);
      return;
    }

    setUploading(true);
    setProgress(0);
    try {
      // 1. Pede URL pré-assinada ao backend
      const presignRes = await fetch(`${API_URL}/admin/uploads/presign`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          filename: file.name,
          content_type: file.type,
          kind: "cover",
        }),
      });
      if (!presignRes.ok) throw new Error("Falha ao obter URL de upload");
      const { upload_url, storage_key, public_url } = await presignRes.json();

      // 2. Faz upload direto pro storage (S3/R2/MinIO), sem passar pela API
      await new Promise<void>((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open("PUT", upload_url);
        xhr.setRequestHeader("Content-Type", file.type);
        xhr.upload.addEventListener("progress", (e) => {
          if (e.lengthComputable) setProgress(Math.round((e.loaded / e.total) * 100));
        });
        xhr.addEventListener("load", () => (xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error(`Upload falhou: ${xhr.status}`))));
        xhr.addEventListener("error", () => reject(new Error("Erro de rede no upload")));
        xhr.send(file);
      });

      // 3. Confirma com o backend (que pode fazer pós-processamento: redimensionar via imgproxy)
      // Opcional — se o backend já devolver public_url no presign, pode pular esta etapa.
      // public_url já aponta para o imgproxy/CDN com a variante "medium".

      onUploaded(public_url, storage_key);
      toast.show("Capa enviada.");
    } catch (e) {
      toast.show(errMsg(e, "Erro no upload da capa"), false);
    } finally {
      setUploading(false);
      setProgress(0);
      if (inputRef.current) inputRef.current.value = "";
    }
  };

  const onInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) handleFile(file);
  };

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file) handleFile(file);
  };

  if (coverUrl && !uploading) {
    return (
      <div className="flex items-start gap-4">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={coverUrl}
          alt="Prévia da capa"
          className="h-40 w-[107px] rounded-lg border border-white/10 object-cover"
        />
        <div className="space-y-2">
          <button
            type="button"
            onClick={() => inputRef.current?.click()}
            className="inline-flex items-center gap-1.5 rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-xs font-medium text-slate-300 hover:bg-white/10 hover:text-white"
          >
            <Icon name="edit" className="h-3.5 w-3.5" /> Trocar capa
          </button>
          <button
            type="button"
            onClick={onRemoved}
            className="block w-full rounded-lg px-3 py-2 text-left text-xs font-medium text-red-300 hover:bg-red-500/10"
          >
            Remover capa
          </button>
          <p className="text-[11px] text-slate-500">A capa será redimensionada automaticamente pelo imgproxy.</p>
        </div>
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPTED.join(",")}
          className="hidden"
          onChange={onInputChange}
        />
      </div>
    );
  }

  return (
    <div
      onDragOver={(e) => {
        e.preventDefault();
        setDragOver(true);
      }}
      onDragLeave={() => setDragOver(false)}
      onDrop={onDrop}
      className={`flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-8 text-center transition-colors ${
        dragOver ? "border-indigo-400/60 bg-indigo-500/10" : "border-white/10 bg-white/[0.02]"
      }`}
    >
      {uploading ? (
        <div className="space-y-3">
          <div className="mx-auto h-10 w-10 animate-spin rounded-full border-2 border-indigo-300/30 border-t-indigo-300" />
          <p className="text-sm text-slate-300">Enviando… {progress}%</p>
          <div className="h-1.5 w-48 overflow-hidden rounded-full bg-white/10">
            <div
              className="h-full bg-gradient-to-r from-indigo-500 to-sky-400 transition-[width] duration-200"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      ) : (
        <>
          <Icon name="image" className="mb-3 h-8 w-8 text-slate-500" />
          <p className="text-sm text-slate-300">
            Arraste uma imagem aqui ou{" "}
            <button
              type="button"
              onClick={() => inputRef.current?.click()}
              className="font-medium text-indigo-300 underline-offset-2 hover:underline"
            >
              selecione um arquivo
            </button>
          </p>
          <p className="mt-1 text-xs text-slate-500">JPG, PNG ou WebP · até 5MB · ideal 2:3</p>
        </>
      )}
      <input
        ref={inputRef}
        type="file"
        accept={ACCEPTED.join(",")}
        className="hidden"
        onChange={onInputChange}
      />
    </div>
  );
}
