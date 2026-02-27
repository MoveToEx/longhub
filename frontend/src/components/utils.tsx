import useAuth from "@/hooks/server/use-auth"
import { useEffect } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";


export function RequiresLogin() {
  const { data, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && !data) {
      toast.info('This page requires you to log in');
      navigate('/');
    }
  }, [data, isLoading, navigate]);
  return <></>
}