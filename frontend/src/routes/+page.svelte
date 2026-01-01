<script lang="ts">
	// 1. 定义数据：使用 Svelte 5 的 $state 让数据变成“活”的
	// 这里的 chatHistory 以后会交给同学 B 来处理逻辑
	let chatHistory = $state([
		{ role: 'assistant', content: '你好！我是 LobeChat 仿制版。我已经准备好为你服务了。' },
		{ role: 'user', content: '太棒了，前端界面看起来不错！' }
	]);

	let inputText = $state(''); // 绑定输入框的文字
</script>

<!-- 整个容器：占据全屏高度 (h-screen)，横向排列 (flex) -->
<div class="flex h-screen w-full bg-gray-50 text-gray-900">
	
	<!-- 左侧边栏：在移动端隐藏 (hidden)，在电脑端显示 (md:flex) -->
	<aside class="hidden w-64 flex-col border-r border-gray-200 bg-white md:flex">
		<div class="border-b p-4 text-xl font-bold text-blue-600">LobeChat Mini</div>
		<nav class="flex-1 overflow-y-auto p-2">
			<div class="mb-2 cursor-pointer rounded-lg bg-blue-50 p-3 text-blue-700">当前对话</div>
			<div class="cursor-pointer rounded-lg p-3 hover:bg-gray-100">历史记录 1</div>
			<div class="cursor-pointer rounded-lg p-3 hover:bg-gray-100">历史记录 2</div>
		</nav>
	</aside>

	<!-- 右侧主聊天区：纵向排列 (flex-col) -->
	<main class="flex flex-1 flex-col">
		<!-- 顶部状态栏 -->
		<header class="flex h-14 items-center border-b bg-white px-4 shadow-sm">
			<div class="flex items-center gap-2">
				<div class="h-3 w-3 rounded-full bg-green-500"></div>
				<span class="font-medium">GPT-4o (智能助手)</span>
			</div>
		</header>

		<!-- 聊天消息展示区：自动撑开 (flex-1)，超出滚动 (overflow-y-auto) -->
		<section class="flex-1 overflow-y-auto p-4 space-y-6">
			{#each chatHistory as msg}
				<div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
					<div class="flex max-w-[85%] gap-3 {msg.role === 'user' ? 'flex-row-reverse' : 'flex-row'}">
						<!-- 头像占位符 -->
						<div class="h-8 w-8 flex-shrink-0 rounded-full bg-gray-300 flex items-center justify-center text-xs">
							{msg.role === 'user' ? 'ME' : 'AI'}
						</div>
						<!-- 气泡内容 -->
						<div class="rounded-2xl px-4 py-2 shadow-sm 
							{msg.role === 'user' ? 'bg-blue-600 text-white' : 'bg-white border border-gray-100'}">
							{msg.content}
						</div>
					</div>
				</div>
			{/each}
		</section>

		<!-- 底部输入区域 -->
		<footer class="border-t bg-white p-4">
			<div class="mx-auto max-w-4xl">
				<div class="relative flex items-end gap-2 rounded-xl border border-gray-200 bg-gray-50 p-2 focus-within:border-blue-400 focus-within:ring-1 focus-within:ring-blue-400">
					<textarea
						bind:value={inputText}
						placeholder="输入消息，Shift + Enter 换行"
						class="max-h-40 w-full resize-none bg-transparent p-2 focus:outline-none"
						rows="1"
					></textarea>
					<button 
						class="rounded-lg bg-black px-4 py-2 text-sm font-medium text-white transition hover:opacity-80 disabled:bg-gray-300"
						disabled={!inputText}
					>
						发送
					</button>
				</div>
				<p class="mt-2 text-center text-xs text-gray-400">由 Svelte 5 + Echo 驱动</p>
			</div>
		</footer>
	</main>
</div>

<style>
	/* 可以在这里写一些微调，但大部分样式我们已经用 Tailwind 搞定了 */
</style>