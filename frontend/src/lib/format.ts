export function formatRupiah(value: number): string {
  return `Rp${new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 }).format(value)}`;
}

export function formatQueueNumber(queueNumber: number): string {
  return `Queue No. ${String(queueNumber).padStart(3, "0")}`;
}
