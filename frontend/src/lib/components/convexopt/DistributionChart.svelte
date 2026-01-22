<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart, registerables } from 'chart.js';
	import type { ConvexPlotData } from '$lib/api/types';

	Chart.register(...registerables);

	let {
		plotData,
		title = 'Distribution',
		useLogScale = false
	}: {
		plotData: ConvexPlotData;
		title?: string;
		useLogScale?: boolean;
	} = $props();

	let canvasRef = $state<HTMLCanvasElement | null>(null);
	let chart: Chart | null = null;

	function createChart() {
		if (!canvasRef || !plotData) return;

		// Destroy existing chart
		if (chart) {
			chart.destroy();
			chart = null;
		}

		const ctx = canvasRef.getContext('2d');
		if (!ctx) return;

		const datasets = [];

		// Theoretical curve (target distribution) - black dashed line
		if (plotData.theoretical_curve && plotData.theoretical_curve.length > 0) {
			datasets.push({
				label: 'Target Distribution',
				data: plotData.theoretical_curve.map((p) => ({ x: p.x, y: p.y })),
				borderColor: '#888888',
				backgroundColor: 'transparent',
				borderDash: [5, 5],
				borderWidth: 2,
				pointRadius: 0,
				tension: 0.3,
				order: 2
			});
		}

		// Solution curve (optimizer result) - red line with markers
		if (plotData.solution_curve && plotData.solution_curve.length > 0) {
			datasets.push({
				label: 'Optimizer Solution',
				data: plotData.solution_curve.map((p) => ({ x: p.x, y: p.y })),
				borderColor: 'rgb(239, 68, 68)', // red-500
				backgroundColor: 'rgba(239, 68, 68, 0.1)',
				borderWidth: 2,
				pointRadius: 3,
				pointBackgroundColor: 'rgb(239, 68, 68)',
				pointBorderColor: 'rgb(239, 68, 68)',
				fill: true,
				tension: 0.1,
				order: 1
			});
		}

		chart = new Chart(ctx, {
			type: 'line',
			data: { datasets },
			options: {
				responsive: true,
				maintainAspectRatio: false,
				interaction: {
					mode: 'nearest',
					intersect: false
				},
				plugins: {
					legend: {
						position: 'top',
						labels: {
							color: '#888',
							font: {
								family: 'ui-monospace, monospace',
								size: 11
							},
							usePointStyle: true,
							pointStyle: 'line'
						}
					},
					tooltip: {
						backgroundColor: 'rgba(0, 0, 0, 0.8)',
						titleFont: {
							family: 'ui-monospace, monospace'
						},
						bodyFont: {
							family: 'ui-monospace, monospace'
						},
						callbacks: {
							label: (context) => {
								const point = context.raw as { x: number; y: number };
								return `${context.dataset.label}: (${point.x.toFixed(2)}, ${point.y.toExponential(3)})`;
							}
						}
					}
				},
				scales: {
					x: {
						type: useLogScale ? 'logarithmic' : 'linear',
						title: {
							display: true,
							text: plotData.x_label || 'Payout Value',
							color: '#666',
							font: {
								family: 'ui-monospace, monospace',
								size: 11
							}
						},
						min: plotData.x_min,
						max: plotData.x_max,
						grid: {
							color: 'rgba(255, 255, 255, 0.05)'
						},
						ticks: {
							color: '#666',
							font: {
								family: 'ui-monospace, monospace',
								size: 10
							}
						}
					},
					y: {
						type: 'linear',
						title: {
							display: true,
							text: plotData.y_label || 'Probability',
							color: '#666',
							font: {
								family: 'ui-monospace, monospace',
								size: 11
							}
						},
						min: 0,
						grid: {
							color: 'rgba(255, 255, 255, 0.05)'
						},
						ticks: {
							color: '#666',
							font: {
								family: 'ui-monospace, monospace',
								size: 10
							},
							callback: (value) => {
								if (typeof value === 'number') {
									if (value === 0) return '0';
									if (value < 0.001) return value.toExponential(1);
									return value.toFixed(4);
								}
								return value;
							}
						}
					}
				}
			}
		});
	}

	onMount(() => {
		createChart();
	});

	onDestroy(() => {
		if (chart) {
			chart.destroy();
			chart = null;
		}
	});

	// Reactively update chart when data changes
	$effect(() => {
		if (plotData && canvasRef) {
			createChart();
		}
	});
</script>

<div class="rounded-xl bg-[var(--color-graphite)]/50 border border-white/[0.03] p-4">
	<div class="flex items-center justify-between mb-3">
		<div class="flex items-center gap-2">
			<div class="w-1 h-4 bg-[var(--color-violet)] rounded-full"></div>
			<h4 class="font-mono text-sm text-[var(--color-light)]">{title}</h4>
		</div>
		<label class="flex items-center gap-2 text-xs text-[var(--color-mist)] cursor-pointer">
			<input
				type="checkbox"
				bind:checked={useLogScale}
				class="w-3 h-3 rounded bg-[var(--color-glass)]"
			/>
			Log scale
		</label>
	</div>

	<div class="relative h-64">
		<canvas bind:this={canvasRef}></canvas>
	</div>

	<!-- Legend explanation -->
	<div class="mt-3 flex items-center justify-center gap-6 text-xs font-mono text-[var(--color-mist)]">
		<div class="flex items-center gap-2">
			<div class="h-0.5 w-5 rounded bg-gray-500" style="background: repeating-linear-gradient(90deg, #888 0, #888 4px, transparent 4px, transparent 8px);"></div>
			<span>Target</span>
		</div>
		<div class="flex items-center gap-2">
			<div class="h-0.5 w-5 rounded bg-red-500"></div>
			<span>Solution</span>
		</div>
	</div>
</div>
