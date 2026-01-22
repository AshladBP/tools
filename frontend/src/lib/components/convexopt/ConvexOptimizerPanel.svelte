<script lang="ts">
	import { api } from '$lib/api/client';
	import { onMount } from 'svelte';
	import type {
		ConvexCriteriaConfig,
		ConvexOptimizerSettings,
		ConvexOptimizeResponse,
		ConvexModeInfoResponse,
		ConvexDistributionType
	} from '$lib/api/types';
	import DistributionChart from './DistributionChart.svelte';

	let {
		mode,
		onOptimize,
		disabled = false
	}: {
		mode: string;
		onOptimize?: (result: ConvexOptimizeResponse) => void;
		disabled?: boolean;
	} = $props();

	// Service availability
	let serviceAvailable = $state<boolean | null>(null);
	let modeInfo = $state<ConvexModeInfoResponse | null>(null);

	// Criteria configuration
	let criteria = $state<ConvexCriteriaConfig[]>([]);
	let optimizerSettings = $state<ConvexOptimizerSettings[]>([]);

	// Global settings
	let weightScale = $state(50);
	let saveToFile = $state(false);
	let createBackup = $state(true);

	// State
	let isLoading = $state(false);
	let result = $state<ConvexOptimizeResponse | null>(null);
	let error = $state<string | null>(null);

	const DISTRIBUTION_TYPES: { value: ConvexDistributionType; label: string }[] = [
		{ value: 'log_normal', label: 'Log-Normal' },
		{ value: 'gaussian', label: 'Gaussian' },
		{ value: 'exponential', label: 'Exponential' }
	];

	onMount(async () => {
		await checkServiceHealth();
		if (serviceAvailable) {
			await loadModeInfo();
		}
	});

	async function checkServiceHealth() {
		try {
			await api.convexHealth();
			serviceAvailable = true;
		} catch {
			serviceAvailable = false;
		}
	}

	async function loadModeInfo() {
		try {
			modeInfo = await api.convexModeInfo(mode);
			// Initialize with default criteria if available
			if (modeInfo.criteria_names.length > 0) {
				addCriteria(modeInfo.criteria_names[0]);
			}
		} catch (e) {
			console.error('Failed to load mode info:', e);
		}
	}

	function addCriteria(name: string = '') {
		criteria = [
			...criteria,
			{
				name: name || `criteria_${criteria.length}`,
				rtp: 0.97,
				hit_rate: 3.0,
				distribution: {
					type: 'log_normal',
					mode: 1.0,
					std: 1.0,
					scale: 1.0
				},
				mix_weight: 1.0
			}
		];
		optimizerSettings = [
			...optimizerSettings,
			{
				kl_divergence_weight: 1.0,
				smoothness_weight: 1.0
			}
		];
	}

	function removeCriteria(index: number) {
		criteria = criteria.filter((_, i) => i !== index);
		optimizerSettings = optimizerSettings.filter((_, i) => i !== index);
	}

	async function runOptimization() {
		if (criteria.length === 0) {
			error = 'Add at least one criteria to optimize';
			return;
		}

		isLoading = true;
		error = null;
		result = null;

		try {
			const response = await api.convexOptimize({
				mode,
				cost: modeInfo?.cost ?? 1.0,
				criteria,
				optimizer_settings: optimizerSettings,
				weight_scale: weightScale,
				lookup_file: modeInfo?.lookup_file ?? `lookUpTable_${mode}.csv`,
				segmented_file: modeInfo?.segmented_file ?? `lookUpTableSegmented_${mode}.csv`,
				win_step_size: 0.1,
				excluded_payouts: [0],
				save_to_file: saveToFile,
				create_backup: createBackup
			});

			result = response;
			onOptimize?.(response);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Optimization failed';
		} finally {
			isLoading = false;
		}
	}

	function formatPercent(value: number): string {
		return (value * 100).toFixed(2) + '%';
	}

	function formatHitRate(value: number): string {
		if (value === Infinity || !isFinite(value)) return 'N/A';
		return `1 in ${value.toFixed(1)}`;
	}
</script>

<div class="space-y-6">
	<!-- Service Unavailable Warning -->
	{#if serviceAvailable === false}
		<div class="px-4 py-3 rounded-xl bg-red-500/10 border border-red-500/30">
			<p class="text-sm text-red-400 font-mono">
				Convex optimizer service is not running.
			</p>
			<p class="text-xs text-[var(--color-mist)] mt-1">
				Start it with: <code class="bg-[var(--color-void)] px-1 rounded">uvicorn src.api.main:app --port 7756</code>
			</p>
		</div>
	{:else if serviceAvailable === null}
		<div class="flex items-center gap-2 text-[var(--color-mist)]">
			<div class="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
			<span class="text-sm">Checking service...</span>
		</div>
	{:else}
		<!-- Header -->
		<div class="flex items-center gap-3">
			<div class="w-1 h-6 bg-[var(--color-violet)] rounded-full"></div>
			<h2 class="font-display text-xl text-[var(--color-light)]">CONVEX OPTIMIZER</h2>
			<span class="text-xs font-mono text-[var(--color-mist)]">{mode}</span>
			{#if modeInfo?.is_bonus_mode}
				<span class="text-xs px-2 py-0.5 rounded bg-[var(--color-gold)]/20 text-[var(--color-gold)]">
					BONUS MODE
				</span>
			{/if}
		</div>

		<!-- Info Banner -->
		<div class="px-4 py-3 rounded-xl bg-[var(--color-violet)]/10 border border-[var(--color-violet)]/30">
			<p class="text-sm text-[var(--color-mist)]">
				CVXPY-based optimization with KL-divergence and smoothness constraints.
				Fits probability distributions to payout data.
			</p>
		</div>

		<!-- Criteria List -->
		{#each criteria as c, i}
			<div class="space-y-4 p-4 rounded-xl border border-[var(--color-glass)] bg-[var(--color-void)]/50">
				<div class="flex items-center justify-between">
					<span class="text-sm font-medium text-[var(--color-light)]">Criteria {i + 1}</span>
					<button
						onclick={() => removeCriteria(i)}
						class="text-xs text-red-400 hover:text-red-300"
						disabled={disabled}
					>
						Remove
					</button>
				</div>

				<div class="grid grid-cols-2 gap-4">
					<!-- Name -->
					<div>
						<label class="block text-xs text-[var(--color-mist)] mb-1">Name</label>
						<input
							type="text"
							bind:value={c.name}
							class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
								text-[var(--color-light)] text-sm focus:outline-none focus:border-[var(--color-violet)]"
							disabled={disabled}
						/>
					</div>

					<!-- Distribution Type -->
					<div>
						<label class="block text-xs text-[var(--color-mist)] mb-1">Distribution</label>
						<select
							bind:value={c.distribution.type}
							class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
								text-[var(--color-light)] text-sm focus:outline-none focus:border-[var(--color-violet)]"
							disabled={disabled}
						>
							{#each DISTRIBUTION_TYPES as dt}
								<option value={dt.value}>{dt.label}</option>
							{/each}
						</select>
					</div>

					<!-- RTP -->
					<div>
						<label class="block text-xs text-[var(--color-mist)] mb-1">Target RTP</label>
						<input
							type="number"
							step="0.01"
							min="0.5"
							max="1.0"
							bind:value={c.rtp}
							class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
								text-[var(--color-light)] text-sm focus:outline-none focus:border-[var(--color-violet)]"
							disabled={disabled}
						/>
					</div>

					<!-- Hit Rate -->
					<div>
						<label class="block text-xs text-[var(--color-mist)] mb-1">Hit Rate (1 in N)</label>
						<input
							type="number"
							step="0.1"
							min="1"
							bind:value={c.hit_rate}
							class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
								text-[var(--color-light)] text-sm focus:outline-none focus:border-[var(--color-violet)]"
							disabled={disabled}
						/>
					</div>
				</div>

				<!-- Distribution Parameters -->
				<div class="grid grid-cols-3 gap-4">
					{#if c.distribution.type === 'log_normal'}
						<div>
							<label class="block text-xs text-[var(--color-mist)] mb-1">Mode</label>
							<input
								type="number"
								step="0.1"
								bind:value={c.distribution.mode}
								class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
									text-[var(--color-light)] text-sm"
								disabled={disabled}
							/>
						</div>
						<div>
							<label class="block text-xs text-[var(--color-mist)] mb-1">Std Dev</label>
							<input
								type="number"
								step="0.1"
								bind:value={c.distribution.std}
								class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
									text-[var(--color-light)] text-sm"
								disabled={disabled}
							/>
						</div>
					{:else if c.distribution.type === 'gaussian'}
						<div>
							<label class="block text-xs text-[var(--color-mist)] mb-1">Mean</label>
							<input
								type="number"
								step="0.1"
								bind:value={c.distribution.mean}
								class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
									text-[var(--color-light)] text-sm"
								disabled={disabled}
							/>
						</div>
						<div>
							<label class="block text-xs text-[var(--color-mist)] mb-1">Std Dev</label>
							<input
								type="number"
								step="0.1"
								bind:value={c.distribution.std}
								class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
									text-[var(--color-light)] text-sm"
								disabled={disabled}
							/>
						</div>
					{:else if c.distribution.type === 'exponential'}
						<div>
							<label class="block text-xs text-[var(--color-mist)] mb-1">Power</label>
							<input
								type="number"
								step="0.1"
								bind:value={c.distribution.power}
								class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
									text-[var(--color-light)] text-sm"
								disabled={disabled}
							/>
						</div>
					{/if}
					<div>
						<label class="block text-xs text-[var(--color-mist)] mb-1">Scale</label>
						<input
							type="number"
							step="0.1"
							bind:value={c.distribution.scale}
							class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
								text-[var(--color-light)] text-sm"
							disabled={disabled}
						/>
					</div>
				</div>

				<!-- Optimizer Settings for this criteria -->
				<div class="grid grid-cols-2 gap-4 pt-2 border-t border-[var(--color-glass)]">
					<div>
						<label class="block text-xs text-[var(--color-mist)] mb-1">
							KL Divergence Weight: {optimizerSettings[i]?.kl_divergence_weight.toFixed(1)}
						</label>
						<input
							type="range"
							min="0.1"
							max="100"
							step="0.1"
							bind:value={optimizerSettings[i].kl_divergence_weight}
							class="w-full"
							disabled={disabled}
						/>
					</div>
					<div>
						<label class="block text-xs text-[var(--color-mist)] mb-1">
							Smoothness: {optimizerSettings[i]?.smoothness_weight.toFixed(1)}
						</label>
						<input
							type="range"
							min="0"
							max="100"
							step="0.5"
							bind:value={optimizerSettings[i].smoothness_weight}
							class="w-full"
							disabled={disabled}
						/>
					</div>
				</div>
			</div>
		{/each}

		<!-- Add Criteria Button -->
		<button
			onclick={() => addCriteria()}
			class="w-full py-2 rounded-lg border border-dashed border-[var(--color-glass)]
				text-[var(--color-mist)] text-sm hover:border-[var(--color-violet)] hover:text-[var(--color-violet)]
				transition-colors"
			disabled={disabled}
		>
			+ Add Criteria
		</button>

		<!-- Global Settings -->
		<div class="grid grid-cols-3 gap-4 p-4 rounded-xl border border-[var(--color-glass)] bg-[var(--color-void)]/50">
			<div>
				<label class="block text-xs text-[var(--color-mist)] mb-1">Weight Scale (2^N)</label>
				<input
					type="number"
					min="30"
					max="64"
					bind:value={weightScale}
					class="w-full px-3 py-2 rounded-lg bg-[var(--color-glass)] border border-[var(--color-glass)]
						text-[var(--color-light)] text-sm"
					disabled={disabled}
				/>
			</div>
			<label class="flex items-center gap-2 cursor-pointer">
				<input
					type="checkbox"
					bind:checked={saveToFile}
					class="w-4 h-4 rounded bg-[var(--color-glass)] border-[var(--color-glass)]"
					disabled={disabled}
				/>
				<span class="text-sm text-[var(--color-mist)]">Save to file</span>
			</label>
			<label class="flex items-center gap-2 cursor-pointer">
				<input
					type="checkbox"
					bind:checked={createBackup}
					class="w-4 h-4 rounded bg-[var(--color-glass)] border-[var(--color-glass)]"
					disabled={disabled}
				/>
				<span class="text-sm text-[var(--color-mist)]">Create backup</span>
			</label>
		</div>

		<!-- Run Button -->
		<button
			onclick={runOptimization}
			disabled={disabled || isLoading || criteria.length === 0}
			class="w-full py-3 rounded-xl bg-[var(--color-violet)] text-[var(--color-light)] font-medium
				hover:bg-[var(--color-violet)]/80 disabled:opacity-50 disabled:cursor-not-allowed
				transition-all"
		>
			{#if isLoading}
				<span class="flex items-center justify-center gap-2">
					<div class="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
					Optimizing...
				</span>
			{:else}
				Run Convex Optimization
			{/if}
		</button>

		<!-- Error Display -->
		{#if error}
			<div class="px-4 py-3 rounded-xl bg-red-500/10 border border-red-500/30">
				<p class="text-sm text-red-400">{error}</p>
			</div>
		{/if}

		<!-- Results -->
		{#if result}
			<div class="space-y-4 p-4 rounded-xl border border-[var(--color-teal)]/30 bg-[var(--color-teal)]/10">
				<div class="flex items-center justify-between">
					<span class="text-sm font-medium text-[var(--color-light)]">Optimization Results</span>
					<span class="text-xs px-2 py-0.5 rounded bg-[var(--color-teal)]/20 text-[var(--color-teal)]">
						{result.success ? 'SUCCESS' : 'FAILED'}
					</span>
				</div>

				<div class="grid grid-cols-3 gap-4 text-sm">
					<div>
						<span class="text-[var(--color-mist)]">Final RTP</span>
						<div class="text-[var(--color-light)] font-mono">{formatPercent(result.final_rtp)}</div>
					</div>
					<div>
						<span class="text-[var(--color-mist)]">Zero Probability</span>
						<div class="text-[var(--color-light)] font-mono">{formatPercent(result.zero_weight_probability)}</div>
					</div>
					<div>
						<span class="text-[var(--color-mist)]">Lookup Length</span>
						<div class="text-[var(--color-light)] font-mono">{result.total_lookup_length}</div>
					</div>
				</div>

				<!-- Criteria Solutions -->
				{#each result.criteria_solutions as sol}
					<div class="p-3 rounded-lg bg-[var(--color-void)]/50 border border-[var(--color-glass)]">
						<div class="flex items-center justify-between mb-2">
							<span class="text-sm font-medium text-[var(--color-light)]">{sol.name}</span>
							<span class="text-xs text-[var(--color-mist)]">{sol.distribution_type}</span>
						</div>
						<div class="grid grid-cols-4 gap-2 text-xs">
							<div>
								<span class="text-[var(--color-mist)]">Target RTP</span>
								<div class="text-[var(--color-light)] font-mono">{formatPercent(sol.target_rtp)}</div>
							</div>
							<div>
								<span class="text-[var(--color-mist)]">Achieved RTP</span>
								<div class="text-[var(--color-light)] font-mono">{formatPercent(sol.achieved_rtp)}</div>
							</div>
							<div>
								<span class="text-[var(--color-mist)]">Target HR</span>
								<div class="text-[var(--color-light)] font-mono">{formatHitRate(sol.target_hit_rate)}</div>
							</div>
							<div>
								<span class="text-[var(--color-mist)]">Achieved HR</span>
								<div class="text-[var(--color-light)] font-mono">{formatHitRate(sol.achieved_hit_rate)}</div>
							</div>
						</div>

						<!-- Distribution Chart -->
						{#if sol.plot_data}
							<div class="mt-4">
								<DistributionChart plotData={sol.plot_data} title="{sol.name} Distribution" />
							</div>
						{/if}
					</div>
				{/each}

				<!-- Warnings -->
				{#if result.warnings && result.warnings.length > 0}
					<div class="p-3 rounded-lg bg-[var(--color-gold)]/10 border border-[var(--color-gold)]/30">
						<span class="text-xs text-[var(--color-gold)]">Warnings:</span>
						<ul class="list-disc list-inside text-xs text-[var(--color-mist)] mt-1">
							{#each result.warnings as warning}
								<li>{warning}</li>
							{/each}
						</ul>
					</div>
				{/if}
			</div>
		{/if}
	{/if}
</div>
