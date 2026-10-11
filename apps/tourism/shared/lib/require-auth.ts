// Sends anonymous visitors to /login once the session check settles. For
// pages that only make sense for a signed-in user (create/edit forms).
export function useRequireAuth(): void {
  const auth = useAuth();
  const localePath = useLocalePath();
  const { isTma, getInitData } = useTelegramWebApp();

  onMounted(async () => {
    if (!auth.checked.value) {
      await auth.ensureSession({ isTma: isTma.value, initData: getInitData() });
    }
    if (!auth.user.value) {
      await navigateTo(localePath('/login'));
    }
  });
}

// Like useRequireAuth, but for pages only approved partners may use:
// everyone else lands on the profile's partner tab, where they can apply.
export function useRequirePartner(): void {
  const auth = useAuth();
  const localePath = useLocalePath();
  const { isTma, getInitData } = useTelegramWebApp();

  onMounted(async () => {
    if (!auth.checked.value) {
      await auth.ensureSession({ isTma: isTma.value, initData: getInitData() });
    }
    if (!auth.user.value) {
      await navigateTo(localePath('/login'));
    } else if (!auth.isPartner.value) {
      await navigateTo(localePath({ path: '/profile', query: { tab: 'partner' } }));
    }
  });
}
