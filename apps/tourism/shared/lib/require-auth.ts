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
