"use client";

import { useEffect, useState, type ChangeEvent, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { avatarSource, ApiError, type User } from "../../lib/api";
import { useAuth } from "../../auth/AuthProvider";

export default function ProfilePage() {
  const router = useRouter();
  const { apiFetch, status, updateProfile, uploadAvatar, user } = useAuth();
  const [profile, setProfile] = useState<User | null>(user);
  const [name, setName] = useState(user?.name ?? "");
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [selectedAvatar, setSelectedAvatar] = useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);

  useEffect(() => () => {
    if (avatarPreview) {
      URL.revokeObjectURL(avatarPreview);
    }
  }, [avatarPreview]);

  useEffect(() => {
    if (status === "anonymous") {
      router.replace("/login");
      return;
    }
    if (status !== "authenticated") return;

    void Promise.resolve(apiFetch<User>("/users/me"))
      .then((currentUser) => {
        setProfile(currentUser);
        setName(currentUser.name);
      })
      .catch(() => setError("Не удалось загрузить профиль. Обновите страницу и попробуйте снова."));
  }, [apiFetch, router, status]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmedName = name.trim();
    setError(null);

    if (!trimmedName) {
      setError("Укажите имя.");
      return;
    }
    if (!profile) return;

    setIsSaving(true);
    try {
      let updatedProfile = profile;
      if (trimmedName !== profile.name) {
        updatedProfile = await updateProfile(trimmedName);
      }
      if (selectedAvatar) {
        updatedProfile = await uploadAvatar(selectedAvatar);
        setSelectedAvatar(null);
        setAvatarPreview(null);
      }
      setProfile(updatedProfile);
      setName(updatedProfile.name);
    } catch (requestError) {
      setError(requestError instanceof ApiError && requestError.status === 400 ? "Проверьте имя и выберите изображение JPEG или PNG." : "Не удалось сохранить изменения. Повторите попытку.");
    } finally {
      setIsSaving(false);
    }
  }

  function handleAvatarSelection(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    if (file.type !== "image/jpeg" && file.type !== "image/png") {
      setError("Поддерживаются только изображения JPEG и PNG.");
      event.target.value = "";
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      setError("Размер изображения не должен превышать 5 МБ.");
      event.target.value = "";
      return;
    }
    setError(null);
    setSelectedAvatar(file);
    setAvatarPreview(URL.createObjectURL(file));
  }

  if (status === "loading") {
    return <main className="grid min-h-screen place-items-center bg-slate-50 text-sm text-slate-500">Загружаем профиль…</main>;
  }
  if (status === "anonymous" || !profile) {
    return null;
  }

  return (
    <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-950 sm:px-8">
      <div className="mx-auto max-w-3xl">
        <header className="flex items-center justify-between">
          <Link className="flex items-center gap-2 font-semibold" href="/dashboard">
            <span className="grid size-8 place-items-center rounded-lg bg-blue-600 text-sm text-white">U</span>
            Uptime
          </Link>
          <Link className="text-sm font-medium text-blue-700 hover:text-blue-800" href="/dashboard">← В кабинет</Link>
        </header>

        <section className="mt-10 rounded-2xl bg-white p-6 shadow-sm ring-1 ring-slate-200 sm:p-8">
          <p className="text-sm font-medium text-blue-700">Настройки аккаунта</p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight">Профиль</h1>
          <p className="mt-2 text-slate-600">Измените имя и фотографию, которые будут отображаться в вашем кабинете.</p>

          <form className="mt-8 max-w-lg space-y-5" onSubmit={handleSubmit} noValidate>
            <fieldset>
              <legend className="text-sm font-medium text-slate-800">Аватар</legend>
              <div className="mt-3 flex items-center gap-4">
                {avatarPreview ? (
                  // Blob URLs are browser-local previews and cannot be optimized by Next.js.
                  // eslint-disable-next-line @next/next/no-img-element
                  <img alt="Предпросмотр аватара" className="size-20 rounded-full border border-slate-200 object-cover" src={avatarPreview} />
                ) : profile.avatarUrl ? (
                  // Avatar files are served by the API and must bypass Next.js image optimization.
                  // eslint-disable-next-line @next/next/no-img-element
                  <img alt="Текущий аватар" className="size-20 rounded-full border border-slate-200 object-cover" src={avatarSource(profile.avatarUrl)} />
                ) : (
                  <span aria-hidden="true" className="grid size-20 place-items-center rounded-full bg-blue-100 text-xl font-semibold text-blue-700">
                    {profile.email.slice(0, 1).toUpperCase()}
                  </span>
                )}
                <div>
                  <label className="inline-flex cursor-pointer rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 transition hover:border-slate-400 hover:bg-slate-50" htmlFor="avatar">
                    Изменить аватар
                  </label>
                  <input accept="image/jpeg,image/png,.jpg,.jpeg,.png" className="sr-only" id="avatar" name="avatar" onChange={handleAvatarSelection} type="file" />
                  <p className="mt-2 text-xs text-slate-500">JPEG или PNG, не более 5 МБ.</p>
                  {selectedAvatar ? <p className="mt-1 text-xs text-slate-600">Выбрано: {selectedAvatar.name}</p> : null}
                </div>
              </div>
            </fieldset>
            <label className="block text-sm font-medium text-slate-800" htmlFor="name">
              Имя
              <input
                autoComplete="name"
                className="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2.5 outline-none ring-blue-600 focus:ring-2"
                id="name"
                maxLength={100}
                name="name"
                onChange={(event) => setName(event.target.value)}
                required
                type="text"
                value={name}
              />
            </label>

            <label className="block text-sm font-medium text-slate-500" htmlFor="email">
              Email
              <input
                className="mt-2 w-full cursor-not-allowed rounded-lg border border-slate-200 bg-slate-50 px-3 py-2.5 text-slate-500"
                disabled
                id="email"
                type="email"
                value={profile.email}
              />
            </label>

            {error ? <p role="alert" className="text-sm text-red-700">{error}</p> : null}
            <button className="rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:bg-blue-400" disabled={isSaving} type="submit">
              {isSaving ? "Сохраняем…" : "Сохранить"}
            </button>
          </form>
        </section>
      </div>
    </main>
  );
}
