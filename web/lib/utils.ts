import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// MIME type mapping for Output node file downloads
const MIME_TYPE_MAP: Record<string, string> = {
  txt: "text/plain",
  md: "text/markdown",
  json: "application/json",
  csv: "text/csv",
  html: "text/html",
  pdf: "application/pdf",
};

/**
 * Trigger a browser file download from a string content.
 * @param content - The file content as a string
 * @param filename - File name without extension (e.g. "rca-report")
 * @param fileType - File extension / type (e.g. "md", "json")
 * @param includeTimestamp - Whether to append a timestamp suffix to the filename
 */
export function downloadFile(
  content: string,
  filename: string = "output",
  fileType: string = "md",
  includeTimestamp: boolean = false,
) {
  const mimeType = MIME_TYPE_MAP[fileType] || "text/plain";
  const ts = includeTimestamp
    ? `-${new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19)}`
    : "";
  const fullName = `${filename}${ts}.${fileType}`;
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = fullName;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
