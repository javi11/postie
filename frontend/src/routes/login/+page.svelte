<script lang="ts">
import { onMount } from "svelte";
import { t } from "$lib/i18n";
import logo from "$lib/assets/images/logo.png";

let registration = $state(false);
let ready = $state(false);
let busy = $state(false);
let username = $state("");
let email = $state("");
let password = $state("");
let error = $state("");

onMount(async () => {
	try {
		const response = await fetch("/api/auth/registration-status", { signal: AbortSignal.timeout(15000) });
		if (!response.ok) throw new Error();
		const status = await response.json();
		registration = status.registration_enabled;
		ready = true;
	} catch { error = $t("common.auth.connectionError"); }
});

async function submit(event: SubmitEvent) {
	event.preventDefault();
	busy = true;
	error = "";
	try {
		const response = await fetch(`/api/auth/${registration ? "register" : "login"}`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ username, email, password }),
		});
		password = "";
		if (!response.ok) {
			error = $t(response.status === 429 ? "common.auth.tooManyAttempts" : registration ? "common.auth.registrationError" : "common.auth.invalidCredentials");
			return;
		}
		window.location.replace("/");
	} catch { error = $t("common.auth.connectionError"); }
	finally { busy = false; }
}
</script>

<svelte:head><title>{$t("common.auth.signIn")} · Postie</title></svelte:head>
<div class="flex min-h-screen items-center justify-center p-6">
	<div class="card w-full max-w-md bg-base-100 shadow-xl">
		<div class="card-body">
			<img src={logo} alt="Postie" class="mx-auto h-16 w-16 object-contain" />
			<h1 class="card-title justify-center">{$t(registration ? "common.auth.createAdmin" : "common.auth.signIn")}</h1>
			{#if registration}<p class="text-sm text-base-content/70">{$t("common.auth.bootstrapDescription")}</p>{/if}
			{#if error}<p class="alert alert-error" role="alert">{error}</p>{/if}
			{#if ready}
				<form onsubmit={submit} class="flex flex-col gap-4">
					<label class="form-control" for="username"><span class="label-text">{$t("common.auth.username")}</span>
						<input id="username" name="username" class="input input-bordered w-full" bind:value={username} autocomplete="username" required minlength={registration ? 3 : undefined} maxlength="128" />
					</label>
					{#if registration}
						<label class="form-control" for="email"><span class="label-text">{$t("common.auth.email")}</span>
							<input id="email" name="email" type="email" class="input input-bordered w-full" bind:value={email} autocomplete="email" maxlength="254" />
						</label>
					{/if}
					<label class="form-control" for="password"><span class="label-text">{$t("common.auth.password")}</span>
						<input id="password" name="password" type="password" class="input input-bordered w-full" bind:value={password} autocomplete={registration ? "new-password" : "current-password"} required minlength={registration ? 12 : undefined} />
					</label>
					{#if registration}<p class="text-xs text-base-content/70">{$t("common.auth.passwordHint")}</p>{/if}
					<button class="btn btn-primary" type="submit" disabled={busy}>{$t(registration ? "common.auth.createAdmin" : "common.auth.signIn")}</button>
				</form>
			{:else if !error}<span class="loading loading-spinner mx-auto"></span>{/if}
		</div>
	</div>
</div>
