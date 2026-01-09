"use client";

import Modal from "@/components/ui/Modal";
import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";

interface DeleteConfirmModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  deviceName: string;
}

export default function DeleteConfirmModal({ isOpen, onClose, onConfirm, deviceName }: DeleteConfirmModalProps) {
  const handleConfirm = () => {
    onConfirm();
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="删除设备" size="sm">
      <div className="text-center">
        <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-rose-500/10 flex items-center justify-center">
          <ExclamationTriangleIcon className="w-8 h-8 text-rose-500" />
        </div>
        <p className="text-slate-700 dark:text-slate-300 mb-2">
          确定要删除设备 <span className="font-semibold text-slate-900 dark:text-white">{deviceName}</span> 吗？
        </p>
        <p className="text-sm text-slate-500 mb-6">
          此操作不可撤销，设备的所有监控数据将被永久删除。
        </p>
        <div className="flex items-center justify-center gap-3">
          <button
            onClick={onClose}
            className="px-6 py-2.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all"
          >
            取消
          </button>
          <button
            onClick={handleConfirm}
            className="px-6 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-rose-500 to-red-600 hover:from-rose-600 hover:to-red-700 rounded-xl shadow-lg shadow-rose-500/30 transition-all"
          >
            确认删除
          </button>
        </div>
      </div>
    </Modal>
  );
}
