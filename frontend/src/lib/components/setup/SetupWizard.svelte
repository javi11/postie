<script lang="ts">
import logo from "$lib/assets/images/logo.png";
import { t } from "$lib/i18n";
import { backend, type config } from "$lib/wailsjs/go/models";
import { Check } from "lucide-svelte";
import DirectorySetupStep from "./DirectorySetupStep.svelte";
import ServerSetupStep from "./ServerSetupStep.svelte";
import WelcomeStep from "./WelcomeStep.svelte";

interface Props {
	oncomplete: (data: backend.SetupWizardData) => void;
}

let { oncomplete }: Props = $props();

let currentStep = $state(1);
let hasValidServers = $state(false);

const stepData = new backend.SetupWizardData();

const steps = [
	{ id: 1, name: $t("setup.steps.welcome"), completed: false },
	{ id: 2, name: $t("setup.steps.servers"), completed: false },
	{ id: 3, name: $t("setup.steps.directories"), completed: false },
];

function nextStep() {
	if (currentStep < steps.length) {
		steps[currentStep - 1].completed = true;
		currentStep++;
	}
}

function prevStep() {
	if (currentStep > 1) {
		currentStep--;
	}
}

// Reactive statement that updates when dependencies change
const canProceed = $derived(
	(() => {
		switch (currentStep) {
			case 1:
				return true; // Welcome step can always proceed
			case 2:
				// For servers step, require at least one server with successful validation
				return hasValidServers;
			case 3:
				return stepData.outputDirectory !== "";
			default:
				return false;
		}
	})(),
);

function handleServerUpdate(event: { servers: backend.ServerData[] }) {
	stepData.servers = event.servers;
}

function handleValidationChange(event: { hasValidServers: boolean }) {
	hasValidServers = event.hasValidServers;
}

function handleDirectoryUpdate(event: {
	outputDirectory: string;
}) {
	stepData.outputDirectory = event.outputDirectory;
}

async function finishSetup() {
	oncomplete(stepData);
}
</script>

<div class="min-h-screen flex flex-col w-full">
	<!-- Header with Logo - Fixed at top -->
	<div class="bg-base-100 border-b border-base-300 py-6">
		<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
			<div class="text-center">
				<div class="flex items-center justify-center gap-3 mb-2">
					<img src={logo} alt="Postie UI" class="w-10 h-10" loading="lazy" />
					<h1 class="text-2xl sm:text-3xl font-bold">
						{$t("setup.wizard_title")}
					</h1>
				</div>
				<p class="text-base-content/70 text-sm sm:text-base">
					{$t("setup.wizard_subtitle")}
				</p>
			</div>
		</div>
	</div>

	<!-- Step Indicator - Fixed below header -->
	<div class="bg-base-200 border-b border-base-300 py-4">
		<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex justify-center overflow-x-auto">
				<div class="flex items-center space-x-4 sm:space-x-8 min-w-max">
					{#each steps as step, index}
						<div class="flex items-center">
							<div class="flex flex-col items-center">
								<div class="flex items-center justify-center w-8 h-8 sm:w-10 sm:h-10 rounded-full border-2 mb-2 transition-all duration-200 {
									step.completed ? 'bg-success border-success text-success-content' :
									currentStep === step.id ? 'bg-primary border-primary text-primary-content' :
									'border-base-300 text-base-content/50'
								}">
									{#if step.completed}
										<Check class="w-4 h-4 sm:w-5 sm:h-5" />
									{:else}
										<span class="text-xs sm:text-sm font-medium">{step.id}</span>
									{/if}
								</div>
								<span class="text-xs sm:text-sm font-medium text-base-content text-center max-w-20 sm:max-w-none">
									{step.name}
								</span>
							</div>
							{#if index < steps.length - 1}
								<div class="w-8 sm:w-16 h-0.5 mx-2 sm:mx-4 transition-colors duration-200 {
									step.completed ? 'bg-success' : 'bg-base-300'
								}"></div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		</div>
	</div>

	<!-- Main Content Area - Scrollable -->
	<div class="flex-1 overflow-y-auto">
		<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
			<div class="card bg-base-100 shadow-lg min-h-[500px] w-full max-w-full">
				<div class="card-body p-6 sm:p-8">
					{#if currentStep === 1}
						<WelcomeStep />
					{:else if currentStep === 2}
						<ServerSetupStep 
							servers={stepData.servers}
							onupdate={handleServerUpdate}
							onvalidationchange={handleValidationChange}
						/>
					{:else if currentStep === 3}
						<DirectorySetupStep 
							outputDirectory={stepData.outputDirectory}
							onupdate={handleDirectoryUpdate}
						/>
					{/if}
				</div>
			</div>
		</div>
	</div>

	<!-- Navigation - Fixed at bottom -->
	<div class="bg-base-100 border-t border-base-300 py-4">
		<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
			<div class="flex justify-between items-center">
				<div>
					{#if currentStep > 1}
						<button class="btn btn-outline btn-sm sm:btn-md" onclick={prevStep}>
							{$t("setup.buttons.previous")}
						</button>
					{:else}
						<div></div>
					{/if}
				</div>
				
				<div class="flex gap-3">
					{#if currentStep < steps.length}
						<button 
							class="btn btn-primary btn-sm sm:btn-md"
							disabled={!canProceed}
							onclick={nextStep}
						>
							{$t("setup.buttons.next")}
						</button>
					{:else}
						<button 
							class="btn btn-success btn-sm sm:btn-md"
							disabled={!canProceed}
							onclick={finishSetup}
						>
							{$t("setup.buttons.finish")}
						</button>
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>