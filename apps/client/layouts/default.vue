<template>
  <div>
    <!-- Navigation Bar -->
    <header class="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex">
            <div class="flex-shrink-0 flex items-center">
              <NuxtLink to="/" class="text-xl font-bold text-gray-900 dark:text-white">
                ClientApp
              </NuxtLink>
            </div>
            <nav class="hidden sm:ml-6 sm:flex sm:space-x-8">
              <NuxtLink to="/" class="border-indigo-500 text-gray-900 dark:text-white inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium">
                Home
              </NuxtLink>
              <NuxtLink to="/about" class="border-transparent text-gray-500 dark:text-gray-400 hover:border-gray-300 dark:hover:border-gray-700 hover:text-gray-700 dark:hover:text-gray-200 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium">
                About
              </NuxtLink>
            </nav>
          </div>
          <div class="hidden sm:ml-6 sm:flex sm:items-center space-x-4">
            <ThemeToggle />
            <UButton to="/login" color="primary" variant="solid">Sign In</UButton>
          </div>
        </div>
      </div>
    </header>

    <!-- Page Content -->
    <main>
      <slot />
    </main>
  </div>
</template>
