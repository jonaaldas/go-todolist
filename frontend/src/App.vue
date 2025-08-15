<template>
  <div class="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800 text-slate-100 p-6">
    <div class="mx-auto max-w-2xl">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-3xl font-bold tracking-tight">Your Todos</h1>
        <div class="flex items-center gap-3">
          <button
            class="inline-flex items-center gap-2 rounded-lg bg-indigo-500 px-4 py-2 text-sm font-medium shadow hover:bg-indigo-400 active:bg-indigo-500/90"
            @click="refresh()">
            <span>Refresh</span>
          </button>
          <button
            class="inline-flex items-center gap-2 rounded-lg bg-indigo-500 px-4 py-2 text-sm font-medium shadow hover:bg-indigo-400 active:bg-indigo-500/90"
            @click="deleteAllTodos()">
            <span>Delete All</span>
          </button>
        </div>
      </div>

      <form
        @submit.prevent="createTodo"
        class="mb-6">
        <div class="flex items-center gap-3">
          <input
            v-model.trim="newTodo"
            type="text"
            placeholder="Add a new todo..."
            class="flex-1 rounded-lg bg-slate-800/60 border border-slate-700 px-4 py-3 text-slate-100 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
          <button
            type="submit"
            class="rounded-lg bg-indigo-500 px-5 py-3 text-sm font-semibold shadow hover:bg-indigo-400 active:bg-indigo-500/90 disabled:opacity-50"
            :disabled="!newTodo">
            Add
          </button>
        </div>
      </form>

      <div
        v-if="todos.length === 0"
        class="text-slate-400 text-center py-16 border border-dashed border-slate-700 rounded-xl">
        No todos yet. Create your first one above.
      </div>

      <ul class="space-y-3">
        <li
          v-for="(todo, idx) in todos"
          :key="todo.id"
          class="group flex items-center gap-3 rounded-xl bg-slate-800/60 border border-slate-700 px-4 py-3">
          <input
            v-model.trim="editable[idx]"
            class="flex-1 bg-transparent outline-none text-slate-100 placeholder-slate-500" />
          <div class="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
            <button
              class="rounded-md px-3 py-2 text-xs font-medium bg-emerald-500 hover:bg-emerald-400 active:bg-emerald-500/90"
              @click="saveEdit(todo.id, editable[idx])">
              Save
            </button>
            <button
              class="rounded-md px-3 py-2 text-xs font-medium bg-rose-500 hover:bg-rose-400 active:bg-rose-500/90"
              @click="removeTodo(todo.id)">
              Delete
            </button>
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ofetch } from 'ofetch'
const todos = ref<{ id: string; todo: string }[]>([])
const newTodo = ref('')
const editable = ref<string[]>([])

const authFetch = ofetch.create({
  headers: {
    user: JSON.stringify({ id: '123', name: 'Jonathan Aldas', email: 'jonaaldas@gmail.com' }),
  },
})

const refresh = async () => {
  const response = await authFetch<{ data: { id: string; todo: string }[] }>('/api/all')
  todos.value = response.data
  editable.value = todos.value.map(t => t.todo)
}

const createTodo = async () => {
  const response = await authFetch<{ success: boolean }>('/api/create', {
    method: 'POST',
    body: { todo: newTodo.value },
  })

  if (response.success) {
    newTodo.value = ''
    refresh()
  }
}

const removeTodo = async (id: string) => {
  console.log('Delete Todo', id)
  const response = await authFetch<{ success: boolean }>('/api/delete/' + id, {
    method: 'DELETE',
  })

  if (response.success) {
    refresh()
  }
}

const saveEdit = async (id: string, text: string) => {
  const response = await authFetch<{ success: boolean }>('/api/edit/' + id, {
    method: 'PUT',
    body: { newUpdatedTodo: text },
  })

  if (response.success) {
    refresh()
  }
}

const deleteAllTodos = async () => {
  const response = await authFetch<{ success: boolean }>('/api/delete/all', {
    method: 'DELETE',
  })

  if (response.success) {
    refresh()
  }
}

onMounted(() => {
  refresh()
})
</script>
<style scoped></style>
