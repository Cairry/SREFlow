"use client";

import React from "react";
import { createPortal } from "react-dom";
import { X, Copy } from "lucide-react";
import { cn } from "@/lib/utils";

interface SidebarDialogProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  nodeId?: string;
  onCopyId?: (id: string) => void;
  children: React.ReactNode;
  className?: string;
}

export function SidebarDialog({
  isOpen,
  onClose,
  title,
  description,
  nodeId,
  onCopyId,
  children,
  className,
}: SidebarDialogProps) {
  if (!isOpen) return null;

  return (
    <div
      className={cn(
        "h-full flex flex-col bg-workflow-node-bg border-workflow-border",
        className
      )}
    >
      <div className="flex items-center justify-between px-4 py-3 border-b border-workflow-border shrink-0">
        <div className="min-w-0 flex-1">
          <h2 className="font-mono text-sm font-semibold text-workflow-text truncate">
            {title}
          </h2>
          {nodeId && (
            <div className="flex items-center gap-1.5 mt-0.5">
              <span className="text-[10px] font-mono text-workflow-text-subtle">
                ID: {nodeId}
              </span>
              <button
                type="button"
                onClick={() => { navigator.clipboard.writeText(nodeId); onCopyId?.(nodeId); }}
                className="text-workflow-text-subtle hover:text-workflow-text transition-colors duration-200 p-0.5 rounded hover:bg-workflow-surface-hover"
                title="Copy ID"
              >
                <Copy className="w-3 h-3" />
              </button>
            </div>
          )}
          {description && (
            <p className="text-xs text-workflow-text-muted mt-0.5 truncate">
              {description}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={onClose}
          className="p-1.5 hover:bg-workflow-surface-hover rounded-lg transition-colors duration-200 text-workflow-text-muted hover:text-workflow-text shrink-0 ml-2"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      <div className="flex-1 overflow-auto">
        <div className="p-4 space-y-3">{children}</div>
      </div>
    </div>
  );
}

// Modal Dialog
interface ModalDialogProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: React.ReactNode;
  className?: string;
  maxHeight?: string;
  width?: "sm" | "md" | "lg" | "xl" | "full";
}

export function ModalDialog({
  isOpen,
  onClose,
  title,
  description,
  children,
  className,
  maxHeight = "max-h-[70vh]",
  width = "md",
}: ModalDialogProps) {
  if (!isOpen) return null;

  const widthClasses: Record<string, string> = {
    sm: "w-80",
    md: "w-96",
    lg: "w-[32rem]",
    xl: "w-[40rem]",
    full: "w-[90vw] h-[85vh]",
  };

  const isFullscreen = width === "full";
  const panelClass = isFullscreen
    ? "fixed inset-4 z-50 rounded-2xl"
    : "fixed inset-0 z-50 flex items-center justify-center mx-4";

  return (
    <>
      {!isFullscreen && (
        <div
          className="absolute inset-0 bg-black/60 backdrop-blur-sm"
          onClick={onClose}
        />
      )}
      <div className={panelClass}>
        <div
          className={cn(
            "relative bg-workflow-surface border border-workflow-border shadow-2xl flex flex-col transition-colors duration-200",
            isFullscreen ? "w-full h-full rounded-2xl" : `${widthClasses[width]!} ${maxHeight} rounded-xl`,
            className
          )}
        >
          <div className="flex items-center justify-between p-4 border-b border-workflow-border shrink-0">
            <div className="min-w-0 flex-1">
              <h2 className="font-mono font-semibold text-workflow-text text-base">{title}</h2>
              {description && <p className="text-xs text-workflow-text-muted mt-0.5">{description}</p>}
            </div>
            <button
              type="button"
              onClick={onClose}
              className="p-1.5 hover:bg-workflow-surface-hover rounded-lg transition-colors duration-200 text-workflow-text-muted shrink-0 ml-2"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
          <div className={cn("flex-1 overflow-auto", isFullscreen ? "p-0" : "p-4")}>{children}</div>
        </div>
      </div>
    </>
  );
}

// Confirm Dialog
interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description?: string;
  confirmText?: string;
  cancelText?: string;
  confirmVariant?: "default" | "destructive";
  isLoading?: boolean;
}

export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  confirmText = "Confirm",
  cancelText = "Cancel",
  confirmVariant = "default",
  isLoading = false,
}: ConfirmDialogProps) {
  if (!isOpen) return null;

  return (
    <ModalDialog isOpen={isOpen} onClose={onClose} title={title} width="sm" maxHeight="auto">
      <div className="space-y-4">
        {description && <div className="text-sm text-workflow-text-muted">{description}</div>}
        <div className="flex justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            className="px-4 py-2 text-sm font-mono text-workflow-text bg-workflow-node-input hover:bg-workflow-surface-hover rounded-lg transition-colors duration-200 disabled:opacity-50"
          >
            {cancelText}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={isLoading}
            className={cn(
              "px-4 py-2 text-sm font-mono text-white rounded-lg transition-colors duration-200 disabled:opacity-50",
              confirmVariant === "destructive" ? "bg-red-500 hover:bg-red-600" : "bg-primary hover:bg-primary/90"
            )}
          >
            {confirmText}
          </button>
        </div>
      </div>
    </ModalDialog>
  );
}

// Form Dialog
interface FormDialogProps extends Omit<ModalDialogProps, "children"> {
  children: React.ReactNode;
  footer?: React.ReactNode;
  onSubmit?: (e: React.FormEvent) => void;
  isLoading?: boolean;
  submitText?: string;
  cancelText?: string;
}

export function FormDialog({
  children,
  footer,
  onSubmit,
  isLoading = false,
  submitText = "Save",
  cancelText = "Cancel",
  onClose,
  ...props
}: FormDialogProps) {
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit?.(e);
  };

  return (
    <ModalDialog onClose={onClose} {...props}>
      <form onSubmit={handleSubmit} className="space-y-4">
        {children}
        {footer && footer}
      </form>
    </ModalDialog>
  );
}

// Menu Dialog
interface MenuDialogProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
  className?: string;
  position?: { x: number; y: number };
}

export function MenuDialog({ isOpen, onClose, children, className, position }: MenuDialogProps) {
  const menuRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose();
      }
    }

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const content = (
    <div
      ref={menuRef}
      className={cn(
        "z-50 bg-workflow-surface border border-workflow-border rounded-lg shadow-xl",
        !position && "absolute",
        className
      )}
      style={position ? {
        position: 'fixed',
        left: Math.min(position.x, window.innerWidth - 340),
        top: position.y,
        maxHeight: Math.min(420, window.innerHeight - position.y - 20),
      } : undefined}
    >
      {children}
    </div>
  );

  if (position) {
    return createPortal(content, document.body);
  }

  return content;
}
