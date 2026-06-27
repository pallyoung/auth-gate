import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

/**
 * Interact with the custom <Select> combobox component.
 *
 * The Select component renders a <button role="combobox"> trigger and
 * a <ul role="listbox"> dropdown that only exists when open. The native
 * <select aria-hidden="true"> is not interactable, so
 * `user.selectOptions()` does not work. This helper clicks the combobox
 * to open the dropdown, then clicks the option with matching text.
 */
export async function selectComboboxOption(
  user: ReturnType<typeof userEvent.setup>,
  comboboxName: string | RegExp,
  optionText: string
) {
  const combobox = screen.getByRole('combobox', { name: comboboxName })
  await user.click(combobox)
  const option = screen.getByRole('option', { name: optionText })
  await user.click(option)
}
