"use client";

import { XMarkIcon } from "@heroicons/react/24/outline";
import { useEffect } from "react";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  size?: "sm" | "md" | "lg" | "xl";
}

const sizeClasses = {
  sm: "max-w-md",
  md: "max-w-lg",
  lg: "max-w-2xl",
  xl: "max-w-4xl",
};

export default function Modal({ isOpen, onClose, title, children, size = "md" }: ModalProps) {
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "unset";
    }
    return () => {
      document.body.style.overflow = "unset";
    };
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* 背景遮罩 */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />
      
      {/* 弹窗内容 */}
      <div className={`relative w-full ${sizeClasses[size]} mx-4 glass-card rounded-2xl shadow-2xl animate-in fade-in zoom-in-95 duration-200`}>
        {/* 头部 */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-white/10">
          <h3 className="text-lg font-semibold text-slate-800 dark:text-white">{title}</h3>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 flex items-center justify-center transition-all"
          >
            <XMarkIcon className="w-5 h-5 text-slate-500 dark:text-slate-400" />
          </button>
        </div>
        
        {/* 内容 */}
        <div className="p-6 max-h-[70vh] overflow-y-auto">
          {children}
        </div>
      </div>
    </div>
  );
}
