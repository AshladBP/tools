<script lang="ts">
	// WelcomeModal is now non-blocking.
	// Certificate trust warnings are shown inline on the LGS tab instead.

	interface Props {
		onReady: () => void;
	}

	let { onReady }: Props = $props();

	// No-op functions for backwards compatibility
	export function show() {}
	export async function backgroundCheck(): Promise<boolean> {
		return true;
	}

	// Immediately signal ready - no blocking modal
	$effect(() => {
		onReady();
	});
</script>

<!-- Modal removed - certificate warnings are now inline on LGS tab -->
