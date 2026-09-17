import Link from "next/link";

export default function Home() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-zinc-50 px-4 text-zinc-900">
      <section className="w-full max-w-xl rounded-2xl bg-white p-10 text-center shadow-sm ring-1 ring-zinc-200">
        <p className="text-sm font-semibold uppercase tracking-[0.2em] text-blue-700">Uptime</p>
        <h1 className="mt-4 text-3xl font-semibold tracking-tight">Мониторинг доступности без лишнего шума</h1>
        <p className="mt-4 text-zinc-600">Создайте аккаунт или войдите, чтобы продолжить.</p>
        <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:justify-center">
          <Link className="rounded-lg bg-blue-700 px-5 py-2.5 font-medium text-white" href="/register">Создать аккаунт</Link>
          <Link className="rounded-lg border border-zinc-300 px-5 py-2.5 font-medium" href="/login">Войти</Link>
        </div>
      </section>
    </main>
  );
}
