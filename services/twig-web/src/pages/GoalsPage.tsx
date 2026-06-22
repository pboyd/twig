import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { listGoals } from "../gen/goal/v1/goal-GoalService_connectquery";
import { AppHeader } from "../components/AppHeader";
import { Spinner } from "../components/Spinner";
import { ErrorBanner } from "../components/ErrorBanner";
import { GoalListItem } from "../components/GoalListItem";
import { goalGroups } from "../lib/goalGroups";
import { readShowHiddenGoals, writeShowHiddenGoals } from "../lib/showHiddenGoalsPref";
import { messages } from "../theme/messages";

export default function GoalsPage() {
  const navigate = useNavigate();
  const [showHidden, setShowHidden] = useState(() => readShowHiddenGoals());

  const { data, isLoading, isError } = useQuery(listGoals, {});

  const groups = useMemo(() => {
    if (!data?.goals) return [];
    return goalGroups(data.goals, showHidden);
  }, [data, showHidden]);

  const totalGoals = data?.goals?.length ?? 0;
  const hasOnlyHidden = totalGoals > 0 && groups.length === 0;

  function toggleHidden() {
    const next = !showHidden;
    setShowHidden(next);
    writeShowHiddenGoals(next);
  }

  return (
    <div className="flex min-h-svh flex-col bg-gray-50 dark:bg-gray-900">
      <AppHeader />
      <main className="flex-1 px-4 py-4">
        {isLoading && (
          <div className="flex justify-center py-12">
            <Spinner size="lg" />
          </div>
        )}

        {isError && (
          <ErrorBanner message={messages.goalLoadError} />
        )}

        {data && groups.length === 0 && !hasOnlyHidden && (
          <div className="flex flex-col items-center justify-center py-16 px-4 text-center gap-4">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {messages.emptyGoalList}
            </p>
          </div>
        )}

        {data && groups.length === 0 && hasOnlyHidden && (
          <div className="flex flex-col items-center justify-center py-16 px-4 text-center gap-4">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {messages.allGoalsHidden}
            </p>
            <button
              onClick={toggleHidden}
              className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
            >
              Show all →
            </button>
          </div>
        )}

        {groups.length > 0 && (
          <div className="space-y-8">
            <div className="flex items-center justify-between">
              <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                {messages.goalsHeading}
              </h1>
              <button
                onClick={toggleHidden}
                className="text-sm font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
              >
                {showHidden ? "Show only active" : "Show all"}
              </button>
            </div>

            {groups.map((group) => (
              <section key={group.state}>
                <h2 className="mb-2 text-sm font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {group.label}
                </h2>
                <div className="flex flex-col gap-2">
                  {group.goals.map((goal) => (
                    <GoalListItem
                      key={goal.id}
                      goal={goal}
                      onOpen={(id) => navigate(`/goals/${id}`)}
                    />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
